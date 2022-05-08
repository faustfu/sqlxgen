package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"go/format"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"os"
	"strconv"
	"strings"
	"unicode"
)

var (
	pUser              *string
	pPassword          *string
	pHost              *string
	pPort              *int
	pDatabase          *string
	pTables            *string
	pPackage           *string
	pJson              *bool
	pGorm              *bool
	pGuregu            *bool
	pSqlx              *bool
	pPath              *string
	pSystemColumns     *string
	pCalculatedColumns *string
)

const (
	DefaultUser              = "root"
	DefaultPassword          = "fr7LwtkL2eWNyuGDchqV4u5h"
	DefaultHost              = "localhost"
	DefaultPort              = 13306
	DefaultDB                = "test_db_01"
	DefaultTables            = "users"
	DefaultJsonFlag          = false
	DefaultGormFlag          = false
	DefaultGureguFlag        = true
	DefaultSqlxFlag          = true
	DefaultPackage           = "dao"
	DefaultPath              = "../.."
	DefaultCreatedColumn     = "created_at"
	DefaultUpdatedColumn     = "updated_at"
	DefaultSystemColumns     = DefaultCreatedColumn + "," + DefaultUpdatedColumn
	DefaultCalculatedColumns = "count(1) count_1 int64"
	DefaultColumnNumber      = 30
)

func main() {
	pUser = flag.String("user", DefaultUser, "DB User")
	pPassword = flag.String("password", DefaultPassword, "DB User Password")
	pHost = flag.String("host", DefaultHost, "DB Host")
	pPort = flag.Int("port", DefaultPort, "DB Port")
	pDatabase = flag.String("database", DefaultDB, "DB Name")
	pTables = flag.String("tables", DefaultTables, "Table Names")
	pJson = flag.Bool("json", DefaultJsonFlag, "Add json tag")
	pGorm = flag.Bool("gorm", DefaultGormFlag, "Add gorm tag")
	pGuregu = flag.Bool("guregu", DefaultGureguFlag, "Use guregu types")
	pSqlx = flag.Bool("sqlx", DefaultSqlxFlag, "Add db tag")
	pPath = flag.String("path", DefaultPath, "Path")
	pPackage = flag.String("package", DefaultPackage, "Package Name")
	pSystemColumns = flag.String("system_columns", DefaultSystemColumns, "Default System Columns")
	pCalculatedColumns = flag.String("calculated_columns", DefaultCalculatedColumns, "Default Calculated Columns")

	flag.Parse()

	calculatedTypes := make(map[string]map[string]string)
	for _, row := range strings.Split(*pCalculatedColumns, ",") {
		types := strings.Split(row, " ")
		formula := types[0]
		n := types[1]
		t := types[2]

		calculatedTypes[n] = make(map[string]string)
		calculatedTypes[n]["formula"] = formula
		calculatedTypes[n]["type"] = t
	}

	for _, tableName := range strings.Split(*pTables, ",") {
		structName := toStructName(tableName)
		fileName := fmt.Sprintf("%s/%s/%s.go", *pPath, *pPackage, structName)

		columnDataTypes, columnsSorted, err := GetColumnsFromMysqlTable(*pUser, *pPassword, *pHost, *pPort, *pDatabase, tableName, *pSystemColumns)

		struc, err := Generate(calculatedTypes, *columnDataTypes, columnsSorted, fmt.Sprintf("`%s`", tableName), structName, *pPackage, *pJson, *pGorm, *pGuregu, *pSqlx)

		if err != nil {
			fmt.Println("Error in creating struct from json: " + err.Error())
			continue
		}

		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println("Open File fail: " + err.Error())
			continue
		}
		length, err := file.WriteString(string(struc))
		if err != nil {
			fmt.Println("Save File fail: " + err.Error())
			continue
		}

		fmt.Printf("wrote %s %d bytes\n", fileName, length)
	}
}

func GetColumnsFromMysqlTable(mariadbUser string, mariadbPassword string, mariadbHost string, mariadbPort int, mariadbDatabase string, mariadbTable string, systemColumns string) (*map[string]map[string]string, []string, error) {
	var err error
	var db *sql.DB
	if mariadbPassword != "" {
		db, err = sql.Open("mysql", mariadbUser+":"+mariadbPassword+"@tcp("+mariadbHost+":"+strconv.Itoa(mariadbPort)+")/"+mariadbDatabase+"?&parseTime=True")
	} else {
		db, err = sql.Open("mysql", mariadbUser+"@tcp("+mariadbHost+":"+strconv.Itoa(mariadbPort)+")/"+mariadbDatabase+"?&parseTime=True")
	}
	defer db.Close()

	// Check for error in db, note this does not check connectivity but does check uri
	if err != nil {
		fmt.Println("Error opening mysql db: " + err.Error())
		return nil, nil, err
	}

	systemColumnNames := strings.Split(systemColumns, ",")
	columnNamesSorted := make([]string, 0, DefaultColumnNumber)

	// Store colum as map of maps
	columnDataTypes := make(map[string]map[string]string)
	// Select columnd data from INFORMATION_SCHEMA
	columnDataTypeQuery := "SELECT COLUMN_NAME, COLUMN_KEY, DATA_TYPE, IS_NULLABLE, COLUMN_COMMENT, EXTRA FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND table_name = ? order by ordinal_position asc"

	rows, err := db.Query(columnDataTypeQuery, mariadbDatabase, mariadbTable)

	if err != nil {
		fmt.Println("Error selecting from db: " + err.Error())
		return nil, nil, err
	}
	if rows != nil {
		defer rows.Close()
	} else {
		return nil, nil, errors.New("No results returned for table")
	}

	for rows.Next() {
		var column string
		var columnKey string
		var dataType string
		var nullable string
		var comment string
		var extra string
		var system string
		rows.Scan(&column, &columnKey, &dataType, &nullable, &comment, &extra)

		for _, v := range systemColumnNames {
			if v == column {
				system = "true"
				break
			}
		}

		columnDataTypes[column] = map[string]string{"value": dataType, "nullable": nullable, "primary": columnKey, "comment": comment, "extra": extra, "system": system}
		columnNamesSorted = append(columnNamesSorted, column)
	}

	return &columnDataTypes, columnNamesSorted, err
}

func Generate(calculatedTypes map[string]map[string]string, columnTypes map[string]map[string]string, columnsSorted []string, tableName string, structName string, pkgName string, jsonAnnotation bool, gormAnnotation bool, gureguTypes bool, sqlxAnnotation bool) ([]byte, error) {

	var dbTypes string

	dbTypes = generateMysqlTypes(calculatedTypes, columnTypes, columnsSorted, tableName, structName, jsonAnnotation, gormAnnotation, gureguTypes, sqlxAnnotation)
	src := fmt.Sprintf("package %s\n%s", pkgName, dbTypes)

	if gormAnnotation == true {
		tableNameFunc := "// TableName sets the insert table name for this struct type\n" +
			"func (" + strings.ToLower(string(structName[0])) + " *" + structName + ") TableName() string {\n" +
			"	return \"" + tableName + "\"" +
			"}"
		src = fmt.Sprintf("%s\n%s", src, tableNameFunc)
	}

	formatted, err := format.Source([]byte(src))
	if err != nil {
		err = fmt.Errorf("error formatting: %s, was formatting\n%s", err, src)
	}

	return formatted, err
}

func generateMysqlTypes(calculatedTypes map[string]map[string]string, obj map[string]map[string]string, columnsSorted []string, tableName string, structName string, jsonAnnotation bool, gormAnnotation bool, gureguTypes bool, sqlxAnnotation bool) string {
	structure := fmt.Sprintf("type %s struct {", structName)

	imports := "\n"
	hasNullable := false // Notice: For guregu types.

	allKeys := make([]string, 0, DefaultColumnNumber)
	pk := "" // Notice: For one PK tables.
	notPKs := make([]string, 0, DefaultColumnNumber)
	insertKeys := make([]string, 0, DefaultColumnNumber)
	insertKeys = append(insertKeys, fmt.Sprintf("`%s`", DefaultCreatedColumn))
	insertValues := make([]string, 0, DefaultColumnNumber)
	insertValues = append(insertValues, "now()")
	updatePairs := make([]string, 0, DefaultColumnNumber)
	updatePairs = append(updatePairs, fmt.Sprintf("`%s` = now()", DefaultUpdatedColumn))

	funcInsert := ""
	funcUpdateAll := ""
	funcExec := ""
	funcQuery := ""
	funcGet := ""

	funcSelectDeleteUpdateAll := ""
	funcSelect := ""
	funcGroup := ""
	funcWhere := ""
	funcHaving := ""
	funcCOMPCriteria := make([]string, 0, DefaultColumnNumber)
	funcAssignCriteria := make([]string, 0, DefaultColumnNumber)
	funcFields := make([]string, 0, DefaultColumnNumber)
	funcNewSQL := ""
	funcGetSQL := ""
	funcOrder := ""
	funcLimit := ""
	funcAscDescSQL := ""
	funcNewOBJ := ""

	for _, key := range columnsSorted {
		mysqlType := obj[key]
		if mysqlType["system"] == "true" {
			continue
		}

		nullable := false
		if mysqlType["nullable"] == "YES" {
			nullable = true
			hasNullable = true
		}

		primary := ""
		if mysqlType["primary"] == "PRI" {
			primary = ";primary_key"
			pk = key
		} else {
			notPKs = append(notPKs, "`"+key+"`")
		}

		// Get the corresponding go value type for this mysql type
		var valueType string
		// If the guregu (https://github.com/guregu/null) CLI option is passed use its types, otherwise use go's sql.NullX

		valueType = mysqlTypeToGoType(mysqlType["value"], nullable, gureguTypes)

		fieldName := fmtFieldName(stringifyFirstChar(key))

		var annotations []string
		if gormAnnotation == true {
			annotations = append(annotations, fmt.Sprintf("gorm:\"column:%s%s\"", key, primary))
		}
		if jsonAnnotation == true {
			annotations = append(annotations, fmt.Sprintf("json:\"%s\"", key))
		}
		if sqlxAnnotation == true {
			annotations = append(annotations, fmt.Sprintf("db:\"%s\"", key))

			funcCOMPCriteria = append(funcCOMPCriteria, fmt.Sprintf(
				"\nfunc (z *%s) C1%s(op string) *%s {\n\tz.sql += \" and `%s` \" + op\n\nreturn z\n}\n",
				structName,
				fieldName,
				structName,
				key,
			))

			funcCOMPCriteria = append(funcCOMPCriteria, fmt.Sprintf(
				"\nfunc (z *%s) C2%s(op string, v %s) *%s {\n\tz.%s = v\n\tz.sql += \" and `%s` \" + op + \" :%s\"\n\nreturn z\n}\n",
				structName,
				fieldName,
				valueType,
				structName,
				fieldName,
				key,
				key,
			))

			funcFields = append(funcFields, fmt.Sprintf(
				"\nfunc (z *%s) Fq%s(suffix string) func() string {\n\treturn func() string {\n\tif suffix == \"\" {\n\t return \"`%s`\"\n}\n\nreturn \"`%s` \" + suffix\n}\n}\n",
				structName,
				fieldName,
				key,
				key,
			))
		}

		if len(annotations) > 0 {
			comment := mysqlType["comment"]
			if comment == "" {
				comment = fieldName
			}
			if primary != "" {
				comment = comment + " (PK)"
			}
			structure += fmt.Sprintf("\n%s %s `%s`  // %s", fieldName, valueType, strings.Join(annotations, " "), comment)
		} else {
			structure += fmt.Sprintf("\n%s %s", fieldName, valueType)
		}

		allKeys = append(allKeys, "`"+key+"`")

		if mysqlType["extra"] == "" { // Skip auto_increment fields as inserting.
			insertKeys = append(insertKeys, "`"+key+"`")
			insertValues = append(insertValues, ":"+key)

			if mysqlType["primary"] != "PRI" { // Skip pk field as updating.
				updatePairs = append(updatePairs, fmt.Sprintf("`%s`=:%s", key, key))
				funcAssignCriteria = append(funcAssignCriteria, fmt.Sprintf(
					"\nfunc (z *%s) Aq%s(v %s) *%s {\n\tz.%s = v\n\tz.sql += \" ,`%s` = :%s\"\n\nreturn z\n}\n",
					structName,
					fieldName,
					valueType,
					structName,
					fieldName,
					key,
					key,
				))
			}
		}
	}
	if sqlxAnnotation {
		for key, v := range calculatedTypes {
			fieldName := fmtFieldName(stringifyFirstChar(key))
			valueType := v["type"]
			formula := v["formula"]

			structure += fmt.Sprintf("\n\t%s %s `db:\"%s\"`", fieldName, valueType, key)

			funcCOMPCriteria = append(funcCOMPCriteria, fmt.Sprintf(
				"\nfunc (z *%s) C1%s(op string) *%s {\n\tz.sql += \" and `%s` \" + op\n\nreturn z\n}\n",
				structName,
				fieldName,
				structName,
				key,
			))

			funcCOMPCriteria = append(funcCOMPCriteria, fmt.Sprintf(
				"\nfunc (z *%s) C2%s(op string, v %s) *%s {\n\tz.%s = v\n\tz.sql += \" and `%s` \" + op + \" :%s\"\n\nreturn z\n}\n",
				structName,
				fieldName,
				valueType,
				structName,
				fieldName,
				key,
				key,
			))

			funcFields = append(funcFields, fmt.Sprintf(
				"\nfunc (z *%s) Fq%s() string {\n\treturn \"%s `%s`\"\n}\n",
				structName,
				fieldName,
				formula,
				key,
			))
		}

		structure += "\n\tc context.Context\n\tdb DB\n\ttx TX\n\tsql string\n}\n"
	} else {
		structure += "\n}\n"
	}

	if sqlxAnnotation {
		// For func:WHERE
		funcWhere = fmt.Sprintf(`
func (z *%s) WHERE() *%s {
	z.sql += " where 1=1 "

	return z
}
`,
			structName,
			structName,
		)

		// For func:HAVING
		funcHaving = fmt.Sprintf(`
func (z *%s) HAVING() *%s {
	z.sql += " having 1=1 "

	return z
}
`,
			structName,
			structName,
		)

		// For func:Insert
		insertSQL := fmt.Sprintf("insert into %s ( %s ) values ( %s )",
			tableName,
			strings.Join(insertKeys, ","),
			strings.Join(insertValues, ","),
		)

		funcInsert = fmt.Sprintf(`
func (z *%s) INSERTAll() *%s {
	z.sql = "%s" // Cannot combine with other sql.

	return z
}
`,
			structName,
			structName,
			insertSQL,
		)

		// for func:UPDATEAll
		updateSQL := fmt.Sprintf("update %s set %s where %s = :%s",
			tableName,
			strings.Join(updatePairs, ","),
			fmt.Sprintf("`%s`", pk),
			pk,
		)

		funcUpdateAll = fmt.Sprintf(`
func (z *%s) UPDATEAll() *%s {
	z.sql = "%s" // Cannot combine with other sql.

	return z
}
`,
			structName,
			structName,
			updateSQL,
		)

		// For func:Exec
		funcExec = fmt.Sprintf(`
func (z *%s) Exec() (affected int64, newID int64, ret error) {
	defer func() {
		if msg := recover(); msg != nil {
			ret = fmt.Errorf(fmt.Sprintf("%%s", msg))
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
`,
			structName,
		)

		// For func:Query
		funcQuery = fmt.Sprintf(`
func (z *%s) Query() ([]*%s, error) {
	rows, err := z.db.NamedQueryContext(z.c, z.sql, z)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	result := make([]*%s, 0)
	for rows.Next() {
		row := New%s(z.c, z.db)
		if err := rows.StructScan(row); err != nil {
			return nil, err
		}
		
		result = append(result, row)
	}
	
	return result, nil
}
`,
			structName,
			structName,
			structName,
			structName,
		)

		// For func:Get
		funcGet = fmt.Sprintf(`
func (z *%s) Get() (*%s, error) {
	rows, err := z.db.NamedQueryContext(z.c, z.sql, z)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	for rows.Next() {
		row := New%s(z.c, z.db)
		if err := rows.StructScan(row); err != nil {
			return nil, err
		}

		return row, nil
	}
	
	return nil, sql.ErrNoRows
}
`,
			structName,
			structName,
			structName,
		)

		// for func:SELECT
		funcSelect = fmt.Sprintf(`
func (z *%s) SELECT(args ...func() string) *%s {
	fields := make([]string, 0, len(args))
	for _, v := range args {
		fields = append(fields, v())
	}

	z.sql += fmt.Sprintf("select %%s from %s", strings.Join(fields, ","))

	return z
}
`,
			structName,
			structName,
			tableName,
		)

		// for func:GROUP
		funcGroup = fmt.Sprintf(`
func (z *%s) GROUP(args ...func() string) *%s {
	fields := make([]string, 0, len(args))
	for _, v := range args {
		fields = append(fields, v())
	}

	z.sql += fmt.Sprintf(" group by %%s", strings.Join(fields, ","))

	return z
}
`,
			structName,
			structName,
		)

		// For func:SelectAll, func:DeleteAll, func:UpdateAll
		funcSelectDeleteUpdateAll = fmt.Sprintf(`
func (z *%s) SELECTAll() *%s {
	z.sql += "select %s from %s "

	return z
}

func (z *%s) UPDATE() *%s {
	z.sql += "update %s set %s = now() "

	return z
}

func (z *%s) DELETE() *%s {
	z.sql += "delete from %s "

	return z
}
`,
			structName,
			structName,
			strings.Join(allKeys, ","),
			tableName,
			structName,
			structName,
			tableName,
			fmt.Sprintf("`%s`", DefaultUpdatedColumn),
			structName,
			structName,
			tableName,
		)

		// for func:NewSQL
		funcNewSQL = fmt.Sprintf(`
func (z *%s) NewSQL(v string) *%s {
	z.sql = v

	return z
}
`,
			structName,
			structName,
		)

		// for func:GetSQL
		funcGetSQL = fmt.Sprintf(`
func (z %s) GetSQL() string {
	return z.sql
}

func (z *%s) Trace() *%s {
	fmt.Println(z.sql)

	return z
}
`,
			structName,
			structName,
			structName,
		)

		// for func:ORDER
		funcOrder = fmt.Sprintf(`
func (z *%s) ORDER(args ...func() string) *%s {
	n := len(args)
	fields := make([]string, n/2)

	for i, v := range args {
		fields[i/2] += v() + " "
	}

	z.sql += fmt.Sprintf(" order by %%s", strings.Join(fields, ","))

	return z
}
`,
			structName,
			structName,
		)

		// for func:LIMIT
		funcLimit = fmt.Sprintf(`
func (z *%s) LIMIT(args ...int64) *%s {
	var offset, limit int64

	n := len(args)
	if n == 1 {
		limit = args[0]
	} else if n == 2 {
		offset = args[0]
		limit = args[1]
	}

	z.sql += fmt.Sprintf(" limit %%d, %%d", offset, limit)

	return z
}
`,
			structName,
			structName,
		)

		// for func:ASC, DESC
		funcAscDescSQL = fmt.Sprintf(`
func (z %s) ASC() string {
	return "asc"
}

func (z %s) DESC() string {
	return "desc"
}
`,
			structName,
			structName,
		)

		// for func:New
		funcNewOBJ = fmt.Sprintf(`
func (z *%s) Init(c context.Context, db DB) *%s {
	z.c = c
	z.db = db
	z.tx = db

	return z
}

func (z *%s) InitX(c context.Context, tx TX) *%s {
	z.c = c
	z.tx = tx

	return z
}

func New%s(c context.Context, db DB) *%s {
	result := &%s{}

	return result.Init(c, db)
}

func New%sX(c context.Context, tx TX) *%s {
	result := &%s{}

	return result.InitX(c, tx)
}
`,
			structName,
			structName,
			structName,
			structName,
			structName,
			structName,
			structName,
			structName,
			structName,
			structName,
		)
	}

	if hasNullable || sqlxAnnotation {
		imports += "import (\n"

		if sqlxAnnotation {
			imports += "\t\"database/sql\"\n"
			imports += "\t\"fmt\"\n"
			imports += "\t\"strings\"\n"
			imports += "\t\"context\"\n"
		}

		if hasNullable && gureguTypes {
			imports += "\t\"gopkg.in/guregu/null.v4\"\n"
		}

		imports += ")\n"
	}

	return imports + structure + funcQuery + funcGet + funcExec +
		funcInsert + funcUpdateAll + funcWhere + funcHaving + funcSelect + funcGroup +
		funcSelectDeleteUpdateAll + strings.Join(funcCOMPCriteria, "\n") +
		strings.Join(funcAssignCriteria, "\n") + strings.Join(funcFields, "\n") +
		funcOrder + funcLimit + funcAscDescSQL + funcNewSQL + funcGetSQL + funcNewOBJ
}

func mysqlTypeToGoType(mysqlType string, nullable bool, gureguTypes bool) string {
	switch mysqlType {
	case "tinyint", "int", "smallint", "mediumint":
		if nullable {
			if gureguTypes {
				return gureguNullInt
			}
			return sqlNullInt
		}
		return golangInt
	case "bigint":
		if nullable {
			if gureguTypes {
				return gureguNullInt
			}
			return sqlNullInt
		}
		return golangInt64
	case "char", "enum", "varchar", "longtext", "mediumtext", "text", "tinytext", "json":
		if nullable {
			if gureguTypes {
				return gureguNullString
			}
			return sqlNullString
		}
		return "string"
	case "date", "datetime", "time", "timestamp":
		if nullable && gureguTypes {
			return gureguNullTime
		}
		return golangTime
	case "decimal", "double":
		if nullable {
			if gureguTypes {
				return gureguNullFloat
			}
			return sqlNullFloat
		}
		return golangFloat64
	case "float":
		if nullable {
			if gureguTypes {
				return gureguNullFloat
			}
			return sqlNullFloat
		}
		return golangFloat32
	case "binary", "blob", "longblob", "mediumblob", "varbinary":
		return golangByteArray
	}
	return ""
}

func fmtFieldName(s string) string {
	if len(s) == 0 {
		return ""
	}

	name := lintFieldName(s)
	runes := []rune(name)
	for i, c := range runes {
		ok := unicode.IsLetter(c) || unicode.IsDigit(c)
		if i == 0 {
			ok = unicode.IsLetter(c)
		}
		if !ok {
			runes[i] = '_'
		}
	}
	return string(runes)
}

func lintFieldName(name string) string {
	// Fast path for simple cases: "_" and all lowercase.
	if name == "_" {
		return name
	}

	for len(name) > 0 && name[0] == '_' {
		name = name[1:]
	}

	allLower := true
	for _, r := range name {
		if !unicode.IsLower(r) {
			allLower = false
			break
		}
	}
	if allLower {
		runes := []rune(name)
		if u := strings.ToUpper(name); commonInitialisms[u] {
			copy(runes[0:], []rune(u))
		} else {
			runes[0] = unicode.ToUpper(runes[0])
		}
		return string(runes)
	}

	// Split camelCase at any lower->upper transition, and split on underscores.
	// Check each word for common initialisms.
	runes := []rune(name)
	w, i := 0, 0 // index of start of word, scan
	for i+1 <= len(runes) {
		eow := false // whether we hit the end of a word

		if i+1 == len(runes) {
			eow = true
		} else if runes[i+1] == '_' {
			// underscore; shift the remainder forward over any run of underscores
			eow = true
			n := 1
			for i+n+1 < len(runes) && runes[i+n+1] == '_' {
				n++
			}

			// Leave at most one underscore if the underscore is between two digits
			if i+n+1 < len(runes) && unicode.IsDigit(runes[i]) && unicode.IsDigit(runes[i+n+1]) {
				n--
			}

			copy(runes[i+1:], runes[i+n+1:])
			runes = runes[:len(runes)-n]
		} else if unicode.IsLower(runes[i]) && !unicode.IsLower(runes[i+1]) {
			// lower->non-lower
			eow = true
		}
		i++
		if !eow {
			continue
		}

		// [w,i) is a word.
		word := string(runes[w:i])
		if u := strings.ToUpper(word); commonInitialisms[u] {
			// All the common initialisms are ASCII,
			// so we can replace the bytes exactly.
			copy(runes[w:], []rune(u))

		} else if strings.ToLower(word) == word {
			// already all lowercase, and not the first word, so uppercase the first character.
			runes[w] = unicode.ToUpper(runes[w])
		}
		w = i
	}
	return string(runes)
}

func stringifyFirstChar(str string) string {
	if len(str) == 0 {
		return ""
	}

	first := str[:1]

	i, err := strconv.ParseInt(first, 10, 8)

	if err != nil {
		return str
	}

	return intToWordMap[i] + "_" + str[1:]
}

func toStructName(v string) string {
	result := ""
	c := cases.Title(language.English)

	rows := strings.Split(v, "_")
	for _, row := range rows {
		result += c.String(row)
	}

	return result
}

const (
	golangByteArray  = "[]byte"
	gureguNullInt    = "null.Int"
	sqlNullInt       = "sql.NullInt64"
	golangInt        = "int"
	golangInt64      = "int64"
	gureguNullFloat  = "null.Float"
	sqlNullFloat     = "sql.NullFloat64"
	golangFloat32    = "float32"
	golangFloat64    = "float64"
	gureguNullString = "null.String"
	sqlNullString    = "sql.NullString"
	gureguNullTime   = "null.Time"
	golangTime       = "time.Time"
)

var commonInitialisms = map[string]bool{
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SSH":   true,
	"TLS":   true,
	"TTL":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
}

var intToWordMap = []string{
	"zero",
	"one",
	"two",
	"three",
	"four",
	"five",
	"six",
	"seven",
	"eight",
	"nine",
}
