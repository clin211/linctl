package authz

import (
	"time"

	casbin "github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
{{- if ne .Component.Storage "mongo" }}
	adapter "github.com/casbin/gorm-adapter/v3"
{{- end }}
	"github.com/google/wire"
{{- if ne .Component.Storage "mongo" }}
	"gorm.io/gorm"
{{- end }}
)

const (
	// defaultAclModel is the inline Casbin RBAC + KeyMatch policy model. It is
	// kept in sync with `configs/casbin/model.conf`; if you customise the file,
	// update this constant accordingly (or remove the constant and have NewAuthz
	// read from the file).
	defaultAclModel = `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act, eft

[role_definition]
g = _, _

[policy_effect]
e = !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && r.act == p.act`

{{- if eq .Component.Storage "mongo" }}

	// defaultPolicyFile is the on-disk Casbin policy used by the file-adapter.
	// Edit `configs/casbin/policy.csv` to grant / revoke permissions; the enforcer
	// reloads it according to autoLoadPolicyTime below.
	defaultPolicyFile = "configs/casbin/policy.csv"
{{- end }}
)

// Authz is a thin wrapper around casbin.SyncedEnforcer that exposes a
// convenient `Authorize(sub, obj, act)` API.
type Authz struct {
	*casbin.SyncedEnforcer
}

// Option is a functional option for NewAuthz.
type Option func(*authzConfig)

// authzConfig collects all configurable knobs for *Authz.
type authzConfig struct {
	aclModel           string        // raw Casbin model
	autoLoadPolicyTime time.Duration // policy auto-reload interval
}

// ProviderSet wires Authz into the Wire DI graph.
var ProviderSet = wire.NewSet(NewAuthz, DefaultOptions)

func defaultAuthzConfig() *authzConfig {
	return &authzConfig{
		aclModel:           defaultAclModel,
		autoLoadPolicyTime: 5 * time.Second,
	}
}

// DefaultOptions returns the default option set for NewAuthz. It can be
// overridden via WithAclModel / WithAutoLoadPolicyTime.
func DefaultOptions() []Option {
	return []Option{
		WithAclModel(defaultAclModel),
		WithAutoLoadPolicyTime(10 * time.Second),
	}
}

// WithAclModel overrides the Casbin model string.
func WithAclModel(model string) Option {
	return func(cfg *authzConfig) {
		cfg.aclModel = model
	}
}

// WithAutoLoadPolicyTime overrides the policy auto-reload interval.
func WithAutoLoadPolicyTime(interval time.Duration) Option {
	return func(cfg *authzConfig) {
		cfg.autoLoadPolicyTime = interval
	}
}

{{- if eq .Component.Storage "mongo" }}

// NewAuthz constructs an *Authz backed by Casbin's built-in file adapter.
//
// MongoDB-backed projects do not have a *gorm.DB to feed gorm-adapter, so the
// scaffold defaults to a file adapter that reads `configs/casbin/policy.csv`.
// Switch to a Mongo Casbin adapter (e.g. casbin-mongo-adapter) if you need
// dynamic policy CRUD.
func NewAuthz(opts ...Option) (*Authz, error) {
	cfg := defaultAuthzConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	m, err := model.NewModelFromString(cfg.aclModel)
	if err != nil {
		return nil, err
	}

	enforcer, err := casbin.NewSyncedEnforcer(m, defaultPolicyFile)
	if err != nil {
		return nil, err
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, err
	}

	enforcer.StartAutoLoadPolicy(cfg.autoLoadPolicyTime)

	return &Authz{enforcer}, nil
}
{{- else }}

// NewAuthz constructs an *Authz backed by Casbin's GORM adapter.
//
// The same DB used by your business store is reused for the Casbin
// policy storage; this gives you transactional policy CRUD with no
// extra infrastructure.
func NewAuthz(db *gorm.DB, opts ...Option) (*Authz, error) {
	cfg := defaultAuthzConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	gormAdapter, err := adapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}

	m, err := model.NewModelFromString(cfg.aclModel)
	if err != nil {
		return nil, err
	}

	enforcer, err := casbin.NewSyncedEnforcer(m, gormAdapter)
	if err != nil {
		return nil, err
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, err
	}

	enforcer.StartAutoLoadPolicy(cfg.autoLoadPolicyTime)

	return &Authz{enforcer}, nil
}
{{- end }}

// Authorize checks whether subject `sub` may perform action `act` on object `obj`.
func (a *Authz) Authorize(sub, obj, act string) (bool, error) {
	return a.Enforce(sub, obj, act)
}
