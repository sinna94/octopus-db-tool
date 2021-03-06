# octopus-db-tools

[![Build Status](https://drone.lechuckcgx.com/api/badges/lechuckroh/octopus-db-tool/status.svg?ref=refs/heads/release/v1)](https://drone.lechuckcgx.com/lechuckroh/octopus-db-tool)

octopus-db-tools provides:
* ERD definition format conversion
* Generate SQL to create tables.
* Generate entity source codes

## Supported Formats
|  format  |input|output|generate|extension|
|---------------------|:-:|:-:|:-:|:------:|
| `octopus`           | O | O |   |`ojson` |
| `xlsx`              | O | O |   |`xlsx`  |
| `staruml2`          | O |   |   |`mdj`   |
| [`dbdiagram.io`][1] |   | O |   |        |
| [`quickdbd`][2]     |   | O |   |        |
| `gorm`              |   |   | O |`go`    |
| `graphql`           |   |   | O |`graphql`, `graphqls`|
| `jpa-kotlin`        |   |   | O |`kt`    |
| `jpa-kotlin-data`   |   |   | O |`kt`    |
| `jpa-groovy`        |   |   |   |`groovy`|
| `jpa-java`          |   |   |   |`java`  |
| `sqlalchemy`        |   |   | O |`py`  |
| `liquibase`         |   |   | O |`yaml`  |
| `opti-studio`       |   |   |   |`xml`   |
| `plantuml`          |   | O |   |`plantuml`|
| `schema-converter`  |   |   |   |`schema`|
| `sql-h2`            |   |   |   |`sql`   |
| `sql-mysql`         | O | O |   |`sql`   |
| `sql-oracle`        |   |   |   |`sql`   |
| `sql-sqlite3`       |   |   |   |`sql`   |
| `sql-sqlserver`     |   |   |   |`sql`   |


[1]: https://dbdiagram.io/
[2]: https://www.quickdatabasediagrams.com/

## Build
### Local Build
Requirements:
* Golang 1.12 or higher
* make

Run:
```bash
$ make vendor
$ make compile

# build os-specific binary
$ make compile-windows
$ make compile-linux
$ make compile-macos
```

### Docker Build
```bash
$ make compile-docker; make compile-rmi
```

## Run
```bash
# create example schema file
$ ./oct create sample.ojson

# convert format
$ ./oct convert <sourceFile> <targetFile> \
    --sourceFormat=<srcFormat> \
    --targetFormat=<targetFormat>

# generate files
$ ./oct generate <sourceFile> <targetFile> \
    --sourceFormat=<srcFormat> \
    --targetFormat=<targetFormat> \
    --package=<packageName> \
    --removePrefix=<prefixes> \
    --uniqueNameSuffix=<suffix>
```

You can omit `--sorceFormat`, `--targetFormat` if file format can be detected.

### Convert
```bash
# starUML2 -> octopus
$ ./oct convert sample.mdj sample.ojson

# octopus -> xlsx
$ ./oct convert sample.ojson sample.xlsx

# octopus -> xlsx (use not null column)
$ ./oct convert sample.ojson sample.xlsx --notNull=true

# xlsx -> octopus
$ ./oct convert sample.xlsx sample.ojson

# octopus -> mysql DDL
$ ./oct convert sample.ojson sample-mysql.sql --targetFormat=mysql

# mysql DDL -> octopus
$ ./oct convert sample-mysql.sql sample.ojson --sourceFormat=mysql
```

#### mysqldump
Octopus does not support all mysql DDL. To generate octopus readable DDL, run the following command :

```bash
$ mysqldump \
    --compact \
    --no-create-db \
    --no-data \
    --skip-add-locks \
    --skip-add-drop-table \
    -u<user> -p -h<host> --databases <DB> \
    > mysql-ddl.sql
```

### Generate
#### octopus -> JPA-kotlin
* entity package: `com.foo.entity`
* repository package: `com.foo.repos`
* graphql package: `com.foo.graphql`
* output directory: `./output`
* remove tableName prefix starting with `db_` or `mydb_`
* unique constraint Name : tableName + `_uq`
* filter table groups: `foo`, `bar`
* add prefix to className: 
    * `foo` group: append `F`
    * `bar` group: append `B`
* add custom class annotations:
    * `foo` group: `@Foo`
    * `foobar` group: `@Foo`, `@Bar`

```bash
$ ./oct generate sample.ojson ./output \
    --targetFormat=jpa-kotlin-data \
    --package=com.foo.entity \
    --reposPackage=com.foo.repos \
    --graphqlPackage=com.foo.graphql \
    --removePrefix=db_,mydb_ \
    --uniqueNameSuffix=_uq \
    --groups=foo,bar,foobar \
    --prefix=foo:F,bar:B \
    --annotation=foo:@Foo,foobar:@Foo;@Bar
```

#### octopus -> SqlAlchemy
* output file: `./output/entities.py`
    * use `./output` to generate separate `*.py` files. 
* remove tableName prefix starting with `db_` or `mydb_`
* unique constraint Name : tableName + `_uq`
* filter table groups: `foo`, `bar`
* add prefix to className: 
    * `foo` group: append `F`
    * `bar` group: append `B`
* use UTC for `created_at`, `updated_at` column default value.

```bash
$ ./oct generate sample.ojson ./output/entities.py \
    --targetFormat=sqlalchemy \
    --removePrefix=db_,mydb_ \
    --uniqueNameSuffix=_uq \
    --groups=foo,bar,foobar
    --prefix=foo:F,bar:B \
    --useUTC=true
```
 
#### octopus -> GORM
* output file: `./output/entities.go`
    * use `./output` to generate separate `*.go` files. 
* remove tableName prefix starting with `db_` or `mydb_`
* unique constraint Name : tableName + `_uq`
* filter table groups: `foo`, `bar`
* add prefix to className: 
    * `foo` group: append `F`
    * `bar` group: append `B`
* custom embedded model: `BaseModel`
    * use `gorm.Model` if not specified.

```bash
$ ./oct generate sample.ojson ./output/entities.go \
    --targetFormat=gorm \
    --removePrefix=db_,mydb_ \
    --uniqueNameSuffix=_uq \
    --groups=foo,bar,foobar \
    --prefix=foo:F,bar:B \
    --gormModel=BaseModel
```
 

#### octopus -> liquibase yaml
Generate all:
* output directory: `./output`
* unique constraint Name : tableName + `_uq`
* generate comments

```bash
$ ./oct generate samples.ojson ./output \
    --targetFormat=liquibase \
    --uniqueNameSuffix=_uq \
    --comments=true
```

Generate diff changelog:
* output directory: `./output`
* unique constraint Name : tableName + `_uq`
* from octopus: `v1.ojson`
* to octopus: `v2.ojson`
* generate comments

```bash
$ ./oct generate v2.ojson ./output \
    --diff=v1.ojson \
    --targetFormat=liquibase \
    --uniqueNameSuffix=_uq \
    --comments=true
```
