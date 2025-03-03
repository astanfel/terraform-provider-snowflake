package sdk

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Grants interface {
	GrantPrivilegeToShare(ctx context.Context, objectPrivilege Privilege, on *GrantPrivilegeToShareOn, to AccountObjectIdentifier) error
	RevokePrivilegeFromShare(ctx context.Context, objectPrivilege Privilege, on *RevokePrivilegeFromShareOn, from AccountObjectIdentifier) error
	Show(ctx context.Context, opts *ShowGrantsOptions) ([]*Grant, error)
}

var _ Grants = (*grants)(nil)

type grants struct {
	client *Client
}

type Grant struct {
	CreatedOn   time.Time
	Privilege   Privilege
	GrantedOn   ObjectType
	Name        ObjectIdentifier
	GrantedTo   ObjectType
	GranteeName AccountObjectIdentifier
	GrantOption bool
	GrantedBy   AccountObjectIdentifier
}

func (v *Grant) ID() ObjectIdentifier {
	return v.Name
}

type grantRow struct {
	CreatedOn   time.Time `db:"created_on"`
	Privilege   string    `db:"privilege"`
	GrantedOn   string    `db:"granted_on"`
	Name        string    `db:"name"`
	GrantedTo   string    `db:"granted_to"`
	GranteeName string    `db:"grantee_name"`
	GrantOption bool      `db:"grant_option"`
	GrantedBy   string    `db:"granted_by"`
}

func (row *grantRow) toGrant() (*Grant, error) {
	grantedTo := ObjectType(row.GrantedTo)
	granteeName := NewAccountObjectIdentifier(row.GranteeName)
	if grantedTo == ObjectTypeShare {
		parts := strings.Split(row.GranteeName, ".")
		name := strings.Join(parts[1:], ".")
		granteeName = NewAccountObjectIdentifier(name)
	}
	grant := &Grant{
		CreatedOn:   row.CreatedOn,
		Privilege:   Privilege(row.Privilege),
		GrantedOn:   ObjectType(row.GrantedOn),
		GrantedTo:   grantedTo,
		Name:        NewAccountObjectIdentifier(strings.Trim(row.Name, "\"")),
		GranteeName: granteeName,
		GrantOption: row.GrantOption,
		GrantedBy:   NewAccountObjectIdentifier(row.GrantedBy),
	}
	return grant, nil
}

type grantPrivilegeToShareOptions struct {
	grant           bool                     `ddl:"static" db:"GRANT"` //lint:ignore U1000 This is used in the ddl tag
	objectPrivilege Privilege                `ddl:"keyword"`
	On              *GrantPrivilegeToShareOn `ddl:"keyword" db:"ON"`
	to              AccountObjectIdentifier  `ddl:"identifier" db:"TO SHARE"`
}

func (opts *grantPrivilegeToShareOptions) validate() error {
	if !validObjectidentifier(opts.to) {
		return ErrInvalidObjectIdentifier
	}
	if !valueSet(opts.On) || opts.objectPrivilege == "" {
		return fmt.Errorf("on and objectPrivilege are required")
	}
	if !exactlyOneValueSet(opts.On.Database, opts.On.Schema, opts.On.Function, opts.On.Table, opts.On.View) {
		return fmt.Errorf("only one of database, schema, function, table, or view can be set")
	}
	return nil
}

type GrantPrivilegeToShareOn struct {
	Database AccountObjectIdentifier `ddl:"identifier" db:"DATABASE"`
	Schema   SchemaIdentifier        `ddl:"identifier" db:"SCHEMA"`
	Function SchemaObjectIdentifier  `ddl:"identifier" db:"FUNCTION"`
	Table    *OnTable                `ddl:"-"`
	View     SchemaObjectIdentifier  `ddl:"identifier" db:"VIEW"`
}

func (v *GrantPrivilegeToShareOn) validate() error {
	if !exactlyOneValueSet(v.Database, v.Schema, v.Function, v.Table, v.View) {
		return fmt.Errorf("only one of database, schema, function, table, or view can be set")
	}
	if valueSet(v.Table) {
		if err := v.Table.validate(); err != nil {
			return err
		}
	}
	return nil
}

type OnTable struct {
	Name        SchemaObjectIdentifier `ddl:"identifier" db:"TABLE"`
	AllInSchema SchemaIdentifier       `ddl:"identifier" db:"ALL TABLES IN SCHEMA"`
}

func (v *OnTable) validate() error {
	if !exactlyOneValueSet(v.Name, v.AllInSchema) {
		return fmt.Errorf("only one of name or allInSchema can be set")
	}
	return nil
}

func (v *grants) GrantPrivilegeToShare(ctx context.Context, objectPrivilege Privilege, on *GrantPrivilegeToShareOn, to AccountObjectIdentifier) error {
	opts := &grantPrivilegeToShareOptions{
		objectPrivilege: objectPrivilege,
		On:              on,
		to:              to,
	}
	if err := opts.validate(); err != nil {
		return err
	}
	sql, err := structToSQL(opts)
	if err != nil {
		return err
	}
	_, err = v.client.exec(ctx, sql)
	return err
}

type revokePrivilegeFromShareOptions struct {
	revoke          bool                        `ddl:"static" db:"REVOKE"` //lint:ignore U1000 This is used in the ddl tag
	objectPrivilege Privilege                   `ddl:"keyword"`
	On              *RevokePrivilegeFromShareOn `ddl:"keyword" db:"ON"`
	from            AccountObjectIdentifier     `ddl:"identifier" db:"FROM SHARE"`
}

func (opts *revokePrivilegeFromShareOptions) validate() error {
	if !validObjectidentifier(opts.from) {
		return ErrInvalidObjectIdentifier
	}
	if !valueSet(opts.On) || opts.objectPrivilege == "" {
		return fmt.Errorf("on and objectPrivilege are required")
	}
	if !exactlyOneValueSet(opts.On.Database, opts.On.Schema, opts.On.Table, opts.On.View) {
		return fmt.Errorf("only one of database, schema, function, table, or view can be set")
	}

	if err := opts.On.validate(); err != nil {
		return err
	}

	return nil
}

type RevokePrivilegeFromShareOn struct {
	Database AccountObjectIdentifier `ddl:"identifier" db:"DATABASE"`
	Schema   SchemaIdentifier        `ddl:"identifier" db:"SCHEMA"`
	Table    *OnTable                `ddl:"-"`
	View     *OnView                 `ddl:"-"`
}

func (v *RevokePrivilegeFromShareOn) validate() error {
	if !exactlyOneValueSet(v.Database, v.Schema, v.Table, v.View) {
		return fmt.Errorf("only one of database, schema, table, or view can be set")
	}
	if valueSet(v.Table) {
		return v.Table.validate()
	}
	if valueSet(v.View) {
		return v.View.validate()
	}
	return nil
}

type OnView struct {
	Name        SchemaObjectIdentifier `ddl:"identifier" db:"VIEW"`
	AllInSchema SchemaIdentifier       `ddl:"identifier" db:"ALL VIEWS IN SCHEMA"`
}

func (v *OnView) validate() error {
	if !exactlyOneValueSet(v.Name, v.AllInSchema) {
		return fmt.Errorf("only one of name or allInSchema can be set")
	}
	return nil
}

func (v *grants) RevokePrivilegeFromShare(ctx context.Context, objectPrivilege Privilege, on *RevokePrivilegeFromShareOn, id AccountObjectIdentifier) error {
	opts := &revokePrivilegeFromShareOptions{
		objectPrivilege: objectPrivilege,
		On:              on,
		from:            id,
	}
	if err := opts.validate(); err != nil {
		return err
	}
	sql, err := structToSQL(opts)
	if err != nil {
		return err
	}
	_, err = v.client.exec(ctx, sql)
	return err
}

type ShowGrantsOptions struct {
	show   bool          `ddl:"static" db:"SHOW"`   //lint:ignore U1000 This is used in the ddl tag
	grants bool          `ddl:"static" db:"GRANTS"` //lint:ignore U1000 This is used in the ddl tag
	On     *ShowGrantsOn `ddl:"keyword" db:"ON"`
	To     *ShowGrantsTo `ddl:"keyword" db:"TO"`
	Of     *ShowGrantsOf `ddl:"keyword" db:"OF"`
}

func (opts *ShowGrantsOptions) validate() error {
	if everyValueNil(opts.On, opts.To, opts.Of) {
		return fmt.Errorf("at least one of on, to, or of is required")
	}
	if !exactlyOneValueSet(opts.On, opts.To, opts.Of) {
		return fmt.Errorf("only one of on, to, or of can be set")
	}
	return nil
}

type ShowGrantsOn struct {
	Account *bool `ddl:"keyword" db:"ACCOUNT"`
	Object  *Object
}

type ShowGrantsTo struct {
	Role  AccountObjectIdentifier `ddl:"identifier" db:"ROLE"`
	User  AccountObjectIdentifier `ddl:"identifier" db:"USER"`
	Share AccountObjectIdentifier `ddl:"identifier" db:"SHARE"`
}

type ShowGrantsOf struct {
	Role  AccountObjectIdentifier `ddl:"identifier" db:"ROLE"`
	Share AccountObjectIdentifier `ddl:"identifier" db:"SHARE"`
}

func (v *grants) Show(ctx context.Context, opts *ShowGrantsOptions) ([]*Grant, error) {
	if opts == nil {
		opts = &ShowGrantsOptions{}
	}
	if err := opts.validate(); err != nil {
		return nil, err
	}
	sql, err := structToSQL(opts)
	if err != nil {
		return nil, err
	}
	var rows []grantRow
	err = v.client.query(ctx, &rows, sql)
	if err != nil {
		return nil, err
	}
	grants := make([]*Grant, 0, len(rows))
	for _, row := range rows {
		grant, err := row.toGrant()
		if err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	return grants, nil
}
