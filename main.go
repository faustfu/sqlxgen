package main

import (
	"context"
	. "faust.link/sqlxgen/dto"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"
	"log"
	"time"
)

func main() {
	db, err := initDB("root:fr7LwtkL2eWNyuGDchqV4u5h@tcp(localhost:13306)/test_db_01?charset=utf8mb4&parseTime=True&multiStatements=true")
	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()

	// delete users
	_, _, err = NewUsers(ctx, db).DELETE().Trace().Exec()
	if err != nil {
		log.Fatalln(err)
	}

	// new user 1
	u := &Users{
		Email:     "u01@example.com",
		FirstName: null.StringFrom("01"),
		LastName:  null.StringFrom("U"),
		Gender:    null.IntFrom(1),
	}
	_, _, err = u.Init(ctx, db).INSERTAll().Trace().Exec()
	if err != nil {
		log.Fatalln(err)
	}

	// new user 2
	u = &Users{
		Email:     "v02@example.com",
		FirstName: null.StringFrom("01"),
		LastName:  null.StringFrom("U"),
		Gender:    null.IntFrom(2),
	}
	_, id, err := u.Init(ctx, db).INSERTAll().Trace().Exec()
	if err != nil {
		log.Fatalln(err)
	}

	// update user 2
	_, _, err = NewUsers(ctx, db).UPDATE().
		AqFirstName(null.StringFrom("02")).
		AqLastName(null.StringFrom("V")).
		WHERE().C2ID(EQ, id).Trace().Exec()
	if err != nil {
		log.Fatalln(err)
	}

	// select users
	rows, err := NewUsers(ctx, db).SELECTAll().Trace().Query()
	if err != nil {
		log.Fatalln(err)
	}
	for _, v := range rows {
		fmt.Println(v.Email, v.FirstName.ValueOrZero(), v.LastName.ValueOrZero())
	}
}

func initDB(url string) (*sqlx.DB, error) {
	if len(url) == 0 {
		return nil, fmt.Errorf("db url is required")
	}

	db, err := sqlx.Connect("mysql", url)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(5 * time.Second)

	return db, nil
}
