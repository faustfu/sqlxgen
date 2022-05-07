package dto

import (
	"context"
	"database/sql"
	"fmt"
	"gopkg.in/guregu/null.v4"
	"strings"
)

type Users struct {
	ID        int64       `db:"id"`         // 流水號 (PK)
	Email     string      `db:"email"`      // 電子郵件帳號
	FirstName null.String `db:"first_name"` // 名
	LastName  null.String `db:"last_name"`  // 姓
	Gender    null.Int    `db:"gender"`     // 性別
	Count1    int64       `db:"count_1"`
	c         context.Context
	db        DB
	tx        TX
	sql       string
}

func (z *Users) Query() ([]*Users, error) {
	rows, err := z.db.NamedQueryContext(z.c, z.sql, z)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	result := make([]*Users, 0)
	for rows.Next() {
		row := NewUsers(z.c, z.db)
		if err := rows.StructScan(row); err != nil {
			return nil, err
		}

		result = append(result, row)
	}

	return result, nil
}

func (z *Users) Get() (*Users, error) {
	rows, err := z.db.NamedQueryContext(z.c, z.sql, z)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	for rows.Next() {
		row := NewUsers(z.c, z.db)
		if err := rows.StructScan(row); err != nil {
			return nil, err
		}

		return row, nil
	}

	return nil, sql.ErrNoRows
}

func (z *Users) Exec() (affected int64, newID int64, ret error) {
	defer func() {
		if msg := recover(); msg != nil {
			ret = fmt.Errorf(fmt.Sprintf("%s", msg))
		}
	}()

	result, err := z.tx.NamedExecContext(z.c, z.sql, z)
	if err != nil || result == nil {
		ret = err
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		ret = err
		return
	}
	affected = rowsAffected

	if affected == 1 {
		lastInsertId, err := result.LastInsertId()
		if err != nil {
			ret = err
			return
		}
		newID = lastInsertId
	}

	return
}

func (z *Users) INSERTAll() *Users {
	z.sql = "insert into `users` ( `created_at`,`email`,`first_name`,`last_name`,`gender` ) values ( now(),:email,:first_name,:last_name,:gender )" // Cannot combine with other sql.

	return z
}

func (z *Users) UPDATEAll() *Users {
	z.sql = "update `users` set `updated_at` = now(),`email`=:email,`first_name`=:first_name,`last_name`=:last_name,`gender`=:gender where `id` = :id" // Cannot combine with other sql.

	return z
}

func (z *Users) WHERE() *Users {
	z.sql += " where 1=1 "

	return z
}

func (z *Users) HAVING() *Users {
	z.sql += " having 1=1 "

	return z
}

func (z *Users) SELECT(args ...func() string) *Users {
	fields := make([]string, 0, len(args))
	for _, v := range args {
		fields = append(fields, v())
	}

	z.sql += fmt.Sprintf("select %s from `users`", strings.Join(fields, ","))

	return z
}

func (z *Users) GROUP(args ...func() string) *Users {
	fields := make([]string, 0, len(args))
	for _, v := range args {
		fields = append(fields, v())
	}

	z.sql += fmt.Sprintf(" group by %s", strings.Join(fields, ","))

	return z
}

func (z *Users) SELECTAll() *Users {
	z.sql += "select `id`,`email`,`first_name`,`last_name`,`gender` from `users` "

	return z
}

func (z *Users) UPDATE() *Users {
	z.sql += "update `users` set `updated_at` = now() "

	return z
}

func (z *Users) DELETE() *Users {
	z.sql += "delete from `users` "

	return z
}

func (z *Users) C1ID(op string) *Users {
	z.sql += " and `id` " + op

	return z
}

func (z *Users) C2ID(op string, v int64) *Users {
	z.ID = v
	z.sql += " and `id` " + op + " :id"

	return z
}

func (z *Users) C1Email(op string) *Users {
	z.sql += " and `email` " + op

	return z
}

func (z *Users) C2Email(op string, v string) *Users {
	z.Email = v
	z.sql += " and `email` " + op + " :email"

	return z
}

func (z *Users) C1FirstName(op string) *Users {
	z.sql += " and `first_name` " + op

	return z
}

func (z *Users) C2FirstName(op string, v null.String) *Users {
	z.FirstName = v
	z.sql += " and `first_name` " + op + " :first_name"

	return z
}

func (z *Users) C1LastName(op string) *Users {
	z.sql += " and `last_name` " + op

	return z
}

func (z *Users) C2LastName(op string, v null.String) *Users {
	z.LastName = v
	z.sql += " and `last_name` " + op + " :last_name"

	return z
}

func (z *Users) C1Gender(op string) *Users {
	z.sql += " and `gender` " + op

	return z
}

func (z *Users) C2Gender(op string, v null.Int) *Users {
	z.Gender = v
	z.sql += " and `gender` " + op + " :gender"

	return z
}

func (z *Users) C1Count1(op string) *Users {
	z.sql += " and `count_1` " + op

	return z
}

func (z *Users) C2Count1(op string, v int64) *Users {
	z.Count1 = v
	z.sql += " and `count_1` " + op + " :count_1"

	return z
}

func (z *Users) AqEmail(v string) *Users {
	z.Email = v
	z.sql += " ,`email` = :email"

	return z
}

func (z *Users) AqFirstName(v null.String) *Users {
	z.FirstName = v
	z.sql += " ,`first_name` = :first_name"

	return z
}

func (z *Users) AqLastName(v null.String) *Users {
	z.LastName = v
	z.sql += " ,`last_name` = :last_name"

	return z
}

func (z *Users) AqGender(v null.Int) *Users {
	z.Gender = v
	z.sql += " ,`gender` = :gender"

	return z
}

func (z *Users) FqID(suffix string) func() string {
	return func() string {
		if suffix == "" {
			return "`id`"
		}

		return "`id` " + suffix
	}
}

func (z *Users) FqEmail(suffix string) func() string {
	return func() string {
		if suffix == "" {
			return "`email`"
		}

		return "`email` " + suffix
	}
}

func (z *Users) FqFirstName(suffix string) func() string {
	return func() string {
		if suffix == "" {
			return "`first_name`"
		}

		return "`first_name` " + suffix
	}
}

func (z *Users) FqLastName(suffix string) func() string {
	return func() string {
		if suffix == "" {
			return "`last_name`"
		}

		return "`last_name` " + suffix
	}
}

func (z *Users) FqGender(suffix string) func() string {
	return func() string {
		if suffix == "" {
			return "`gender`"
		}

		return "`gender` " + suffix
	}
}

func (z *Users) FqCount1() string {
	return "count(1) `count_1`"
}

func (z *Users) ORDER(args ...func() string) *Users {
	n := len(args)
	fields := make([]string, n/2)

	for i, v := range args {
		fields[i/2] += v() + " "
	}

	z.sql += fmt.Sprintf(" order by %s", strings.Join(fields, ","))

	return z
}

func (z *Users) LIMIT(args ...int64) *Users {
	var offset, limit int64

	n := len(args)
	if n == 1 {
		limit = args[0]
	} else if n == 2 {
		offset = args[0]
		limit = args[1]
	}

	z.sql += fmt.Sprintf(" limit %d, %d", offset, limit)

	return z
}

func (z Users) ASC() string {
	return "asc"
}

func (z Users) DESC() string {
	return "desc"
}

func (z *Users) NewSQL(v string) *Users {
	z.sql = v

	return z
}

func (z Users) GetSQL() string {
	return z.sql
}

func (z *Users) Trace() *Users {
	fmt.Println(z.sql)

	return z
}

func (z *Users) Init(c context.Context, db DB) *Users {
	z.c = c
	z.db = db
	z.tx = db

	return z
}

func (z *Users) InitX(c context.Context, tx TX) *Users {
	z.c = c
	z.tx = tx

	return z
}

func NewUsers(c context.Context, db DB) *Users {
	result := &Users{}

	return result.Init(c, db)
}

func NewUsersX(c context.Context, tx TX) *Users {
	result := &Users{}

	return result.InitX(c, tx)
}
