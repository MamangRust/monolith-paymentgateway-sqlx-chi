package tests

import (
	roleadapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/role"
	useradapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter/user"
	"github.com/MamangRust/monolith-payment-gateway-pkg/hash"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/MamangRust/monolith-payment-gateway-shared/cache"
	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbrole "github.com/MamangRust/monolith-payment-gateway-pb/role"
	pbuser "github.com/MamangRust/monolith-payment-gateway-pb/user"
	role_handler "github.com/MamangRust/monolith-payment-gateway-role/handler"
	role_repo "github.com/MamangRust/monolith-payment-gateway-role/repository"
	role_service "github.com/MamangRust/monolith-payment-gateway-role/service"
	user_handler "github.com/MamangRust/monolith-payment-gateway-user/handler"
	user_repo "github.com/MamangRust/monolith-payment-gateway-user/repository"
	user_service "github.com/MamangRust/monolith-payment-gateway-user/service"
)

// RoleGRPC bundles an in-process role gRPC service with the role adapters that
// other services depend on. Services such as user and auth resolve ROLE_ADMIN
// and assign it to new users through these adapters.
type RoleGRPC struct {
	Conn            *grpc.ClientConn
	Server          *grpc.Server
	RoleAdapter     roleadapter.RoleAdapter
	UserRoleAdapter roleadapter.UserRoleAdapter
}

// StartRoleGRPC starts an in-process role service backed by db. The caller owns
// the returned resources and must call Close when finished.
func StartRoleGRPC(db *sqlx.DB, cacheStore *cache.CacheStore, log logger.LoggerInterface) (*RoleGRPC, error) {
	svc := role_service.NewService(&role_service.Deps{
		Cache:        cacheStore,
		Repositories: role_repo.NewRepositories(db),
		Logger:       log,
	})
	h := role_handler.NewHandler(svc)

	server := grpc.NewServer()
	pbrole.RegisterRoleServiceServer(server, h.RoleQuery)
	pbrole.RegisterRoleCommandServiceServer(server, h.RoleCommand)

	addr, err := RunGRPCServer(server)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		server.Stop()
		return nil, err
	}

	return &RoleGRPC{
		Conn:            conn,
		Server:          server,
		RoleAdapter:     roleadapter.NewRoleAdapter(pbrole.NewRoleServiceClient(conn)),
		UserRoleAdapter: roleadapter.NewUserRoleAdapter(pbrole.NewRoleCommandServiceClient(conn)),
	}, nil
}

func (r *RoleGRPC) Close() {
	if r == nil {
		return
	}
	if r.Conn != nil {
		_ = r.Conn.Close()
	}
	if r.Server != nil {
		r.Server.Stop()
	}
}

// UserGRPC bundles an in-process user gRPC service with a user adapter. The
// service is wired to the role dependency so user creation can assign the
// default ROLE_ADMIN.
type UserGRPC struct {
	Conn    *grpc.ClientConn
	Server  *grpc.Server
	Service user_service.Service
	Adapter useradapter.QueryRepository
}

// StartUserGRPC starts an in-process user service backed by db and role.
func StartUserGRPC(db *sqlx.DB, cacheStore *cache.CacheStore, log logger.LoggerInterface, role *RoleGRPC) (*UserGRPC, error) {
	svc := user_service.NewService(&user_service.Deps{
		Cache: cacheStore,
		Repositories: user_repo.NewRepositories(
			db,
			pbrole.NewRoleServiceClient(role.Conn),
			pbrole.NewRoleCommandServiceClient(role.Conn),
		),
		Hash:            hash.NewHashingPassword(),
		Logger:          log,
		RoleAdapter:     role.RoleAdapter,
		UserRoleAdapter: role.UserRoleAdapter,
	})
	h := user_handler.NewHandler(svc)

	server := grpc.NewServer()
	pbuser.RegisterUserQueryServiceServer(server, h)
	pbuser.RegisterUserCommandServiceServer(server, h)

	addr, err := RunGRPCServer(server)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		server.Stop()
		return nil, err
	}

	return &UserGRPC{
		Conn:    conn,
		Server:  server,
		Service: svc,
		Adapter: useradapter.NewQueryAdapter(pbuser.NewUserQueryServiceClient(conn)),
	}, nil
}

func (u *UserGRPC) Close() {
	if u == nil {
		return
	}
	if u.Conn != nil {
		_ = u.Conn.Close()
	}
	if u.Server != nil {
		u.Server.Stop()
	}
}

// IdentityStack is the role + user dependency pair required by services that
// validate users or assign roles. It is the test equivalent of the gRPC
// dependencies wired in each service's apps/server.go.
type IdentityStack struct {
	Role *RoleGRPC
	User *UserGRPC
}

// StartIdentityStack starts a role service and a user service wired to it.
func StartIdentityStack(db *sqlx.DB, cacheStore *cache.CacheStore, log logger.LoggerInterface) (*IdentityStack, error) {
	role, err := StartRoleGRPC(db, cacheStore, log)
	if err != nil {
		return nil, err
	}

	user, err := StartUserGRPC(db, cacheStore, log, role)
	if err != nil {
		role.Close()
		return nil, err
	}

	return &IdentityStack{Role: role, User: user}, nil
}

func (s *IdentityStack) Close() {
	if s == nil {
		return
	}
	s.User.Close()
	s.Role.Close()
}
