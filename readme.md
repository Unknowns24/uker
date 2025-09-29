# Uker

[![Último release](https://img.shields.io/github/release/unknowns24/uker.svg)](https://github.com/unknowns24/uker/releases)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/unknowns24/uker)

## Tabla de contenidos

- [Descripción general](#descripción-general)
- [Requisitos](#requisitos)
- [Instalación](#instalación)
- [Guía rápida de uso](#guía-rápida-de-uso)
  - [Cargar configuración desde variables de entorno](#cargar-configuración-desde-variables-de-entorno)
  - [Conectar a MySQL con GORM](#conectar-a-mysql-con-gorm)
  - [Procesar peticiones HTTP](#procesar-peticiones-http)
  - [Validaciones y manejo de errores](#validaciones-y-manejo-de-errores)
  - [Paginación basada en cursores](#paginación-basada-en-cursor)
  - [Logging centralizado con Fluentd](#logging-centralizado-con-fluentd)
  - [Otras utilidades](#otras-utilidades)
- [Ejemplo de integración en un servicio HTTP](#ejemplo-de-integración-en-un-servicio-http)
- [Recursos adicionales](#recursos-adicionales)

## Descripción general

Uker (Utility Kernel) es un conjunto de utilidades escritas en Go pensado para acelerar la construcción de servicios. El módulo incluye ayudas para cargar configuración, abrir conexiones a base de datos, procesar peticiones HTTP, validar datos, manejar errores, generar identificadores, loggear hacia Fluentd y paginar resultados con cursores.

La librería está organizada en subpaquetes que puedes importar de forma selectiva según lo necesites:

```
github.com/unknowns24/uker/uker/config
github.com/unknowns24/uker/uker/db
github.com/unknowns24/uker/uker/errors
github.com/unknowns24/uker/uker/fn
github.com/unknowns24/uker/uker/httpx
github.com/unknowns24/uker/uker/id
github.com/unknowns24/uker/uker/log
github.com/unknowns24/uker/uker/pagination
github.com/unknowns24/uker/uker/validate
```

## Requisitos

- Go 1.24 o superior (según lo declarado en `go.mod`).
- Para el paquete `db`, necesitas `gorm.io/gorm` y `gorm.io/driver/mysql` (se instalan automáticamente como dependencias transitivas).
- Para el paquete `log`, debes tener un servicio Fluentd accesible desde tu aplicación.

## Instalación

Agrega el módulo a tu proyecto con `go get`:

```bash
go get github.com/unknowns24/uker
```

Luego importa los paquetes que necesites. Por ejemplo:

```go
import (
    "github.com/unknowns24/uker/uker/config"
    "github.com/unknowns24/uker/uker/httpx"
)
```

## Guía rápida de uso

### Cargar configuración desde variables de entorno

El cargador lee estructuras etiquetadas con `config:""` y antepone un prefijo para buscar variables en mayúsculas.

```go
package main

import (
    "fmt"

    "github.com/unknowns24/uker/uker/config"
)

type AppConfig struct {
    Host string `config:"host"`
    Port string `config:"port"`
}

func main() {
    var cfg AppConfig

    loader := config.New("API")
    if err := loader.Load(&cfg); err != nil {
        panic(err)
    }

    fmt.Printf("Servidor en %s:%s\n", cfg.Host, cfg.Port)
}
```

Este ejemplo espera variables de entorno `API_HOST` y `API_PORT`. Si no defines etiqueta `config`, el nombre del campo se usa como clave.

### Conectar a MySQL con GORM

El conector crea la cadena de conexión y puede ejecutar migraciones automáticamente.

```go
import (
    "log"

    "github.com/unknowns24/uker/uker/db"
    "gorm.io/gorm"
)

type User struct {
    ID    uint
    Email string
}

func openDatabase() *gorm.DB {
    connector := db.NewMySQL(db.MySQLConnData{
        Host:     "127.0.0.1",
        Port:     "3306",
        Database: "app",
        User:     "root",
        Password: "secret",
    })

    conn, err := connector.Open(&User{})
    if err != nil {
        log.Fatalf("no se pudo abrir la base: %v", err)
    }

    return conn
}
```

### Procesar peticiones HTTP

`httpx` incluye helpers para parsear cuerpos JSON y formularios multipart, aplicando validaciones automáticas basadas en tags `uker:"required"`.

```go
import (
    "net/http"

    "github.com/unknowns24/uker/uker/httpx"
)

type CreateUserRequest struct {
    Email string `json:"email" uker:"required"`
    Name  string `json:"name"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := httpx.BodyParser(r, &req); err != nil {
        httpx.ErrorOutput(w, http.StatusBadRequest, httpx.Response{
            Status: httpx.ResponseStatus{Type: httpx.Error, Code: "bad_request", Description: err.Error()},
        })
        return
    }

    httpx.FinalOutput(w, http.StatusCreated, httpx.Response{
        Status: httpx.ResponseStatus{Type: httpx.Success, Code: "user_created"},
        Data:   req,
    })
}
```

Para payloads codificados en base64, añade la opción `httpx.WithBase64Data()`.

### Validaciones y manejo de errores

Usa `validate` para comprobaciones simples y `errors` para envolver errores de dominio con códigos legibles.

```go
import (
    liberr "github.com/unknowns24/uker/uker/errors"
    "github.com/unknowns24/uker/uker/validate"
)

func validateEmail(email string) error {
    if err := validate.NotEmpty(email); err != nil {
        return liberr.New("empty_email", err.Error())
    }
    return nil
}
```

`validate.RequiredFields` se usa internamente en `httpx` y puedes invocarlo manualmente si decodificas JSON por tu cuenta.

### Paginación basada en cursor

El paquete `pagination` implementa un contrato consistente para consultas en cursor.

```go
import (
    "net/http"
    "time"

    "github.com/unknowns24/uker/uker/httpx"
    "github.com/unknowns24/uker/uker/pagination"
)

var cursorSecret = []byte("super-secret")

// db es un *gorm.DB inicializado en otro lugar del servicio.

func listUsers(w http.ResponseWriter, r *http.Request) {
    params, err := pagination.ParseWithSecurity(r.URL.Query(), cursorSecret, time.Hour)
    if err != nil {
        httpx.ErrorOutput(w, http.StatusBadRequest, httpx.Response{
            Status: httpx.ResponseStatus{Type: httpx.Error, Code: "invalid_cursor", Description: err.Error()},
        })
        return
    }

    query, err := pagination.Apply(db.Model(&User{}), params)
    if err != nil {
        httpx.ErrorOutput(w, http.StatusBadRequest, httpx.Response{
            Status: httpx.ResponseStatus{Type: httpx.Error, Code: "invalid_query", Description: err.Error()},
        })
        return
    }

    var rows []User
    if err := query.Find(&rows).Error; err != nil {
        httpx.ErrorOutput(w, http.StatusInternalServerError, httpx.Response{
            Status: httpx.ResponseStatus{Type: httpx.Error, Code: "db_error", Description: err.Error()},
        })
        return
    }

    page, err := pagination.BuildPageSigned(params, rows, params.Limit, nil, cursorSecret)
    if err != nil {
        httpx.ErrorOutput(w, http.StatusInternalServerError, httpx.Response{
            Status: httpx.ResponseStatus{Type: httpx.Error, Code: "cursor_error", Description: err.Error()},
        })
        return
    }

    httpx.FinalOutput(w, http.StatusOK, httpx.Response{Status: httpx.ResponseStatus{Type: httpx.Success, Code: "ok"}, Data: page})
}
```

Notas clave del módulo:

- `Apply` consulta `limit+1` registros para determinar `has_more` sin lecturas adicionales.
- `ParseWithSecurity` y `BuildPageSigned` emiten y verifican cursores firmados con HMAC y TTL configurable.
- Los identificadores de filtros y orden se validan con regex y whitelist opcional (`pagination.AllowedColumns`).
- Si una petición incluye `cursor`, los filtros y orden no pueden modificarse en la querystring.
- Los filtros `*_like` aplican `%valor%` para búsquedas de “contiene”.
- Asegúrate de que las columnas usadas para ordenar no admitan `NULL` o documenta la limitación.
- Recomendación de índices compuestos (ejemplo): `CREATE INDEX idx_users_status_created_id ON users (status, created_at DESC, id DESC);`
- En Postgres/MySQL 8+, las comparaciones por tuplas (`(created_at, id)`) aceleran el keyset cuando el orden es uniforme.

### Logging centralizado con Fluentd

El paquete `log` expone un wrapper de `logrus` que envía eventos a Fluentd con reintentos y backoff exponencial.

```go
import (
    "time"

    "github.com/fluent/fluent-logger-golang/fluent"
    "github.com/sirupsen/logrus"
    ulog "github.com/unknowns24/uker/uker/log"
)

func newLogger() *ulog.Logger {
    cfg := &ulog.Config{
        FluentMetadata: ulog.FluentMetadata{
            Tag:         "app.events",
            Source:      "api",
            ServiceName: "users",
            Application: "admin",
        },
        FluentConfig: fluent.Config{FluentPort: 24224, FluentHost: "fluentd"},
        LogFormatter: &logrus.JSONFormatter{},
        LogOnConsole: true,
        TestConnectionTime: 30 * time.Second,
    }

    logger := ulog.New(cfg)
    logger.Logger.Info("logger inicializado")
    return logger
}
```

La función `RetryWithBackoff` permite reintentar operaciones con un backoff exponencial aleatorizado.

### Otras utilidades

- `id` genera identificadores hexadecimales (`id.MustNew()`) o más cortos, seguros para URLs (`id.Short()`).
- `fn` contiene ayudas genéricas para slices, maps y strings (`fn.Map`, `fn.Keys`, `fn.Sanitize`, etc.).
- `pagination.EncodeCursor` y `pagination.DecodeCursor` te permiten firmar y leer cursores sin acoplarte a HTTP.
- El paquete raíz define `uker.Version`, útil para exponer la versión del módulo en endpoints de health-check.

## Ejemplo de integración en un servicio HTTP

```go
package main

import (
    "log"
    "net/http"

    "github.com/unknowns24/uker/uker/config"
    "github.com/unknowns24/uker/uker/db"
    "github.com/unknowns24/uker/uker/httpx"
    "github.com/unknowns24/uker/uker/pagination"
)

type AppConfig struct {
    DBHost string `config:"db_host"`
    DBPort string `config:"db_port"`
}

func main() {
    var cfg AppConfig
    if err := config.New("APP").Load(&cfg); err != nil {
        log.Fatal(err)
    }

    connector := db.NewMySQL(db.MySQLConnData{
        Host:     cfg.DBHost,
        Port:     cfg.DBPort,
        Database: "app",
        User:     "root",
        Password: "secret",
    })
    conn, err := connector.Open()
    if err != nil {
        log.Fatal(err)
    }
    _ = conn // úsalo para tus repositorios

    http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
        params, err := pagination.Parse(r.URL.Query())
        if err != nil {
            httpx.ErrorOutput(w, http.StatusBadRequest, httpx.Response{
                Status: httpx.ResponseStatus{Type: httpx.Error, Code: "invalid_pagination", Description: err.Error()},
            })
            return
        }

        // fetchUsers aplica params para construir la consulta y devolver resultados.
        users := fetchUsers(params)
        httpx.FinalOutput(w, http.StatusOK, pagination.NewPage(users, params.Limit, false, "", ""))
    })

    log.Println("Servidor en :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func fetchUsers(p pagination.Params) []string {
    return []string{"alice@example.com", "bob@example.com"}
}
```

Este es un punto de partida: añade tus repositorios, validaciones extra y middleware favorito.

## Recursos adicionales

- [Documentación en pkg.go.dev](https://pkg.go.dev/github.com/unknowns24/uker)
- [Releases publicados](https://github.com/unknowns24/uker/releases)
- [Reporta issues o propone mejoras](https://github.com/unknowns24/uker/issues)

Para conocer la versión del módulo en tiempo de ejecución, revisa `uker.Version`.
