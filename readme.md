# Uker

## Overview

Uker (Utility Kernel) is a Go package that provides a set of utilities to simplify various aspects of programming. This package includes interfaces that cover common functionality in the areas of gRPC, HTTP, Middlewares, and MySQL database interactions. These utilities aim to make your Go programming tasks more efficient and straightforward.

## Table of Contents

-   [Installation](#installation)
-   [Usage](#usage)
-   [Interfaces](#interfaces)
    -   [gRPC Interface](#grpc-interface)
    -   [HTTP Interface](#http-interface)
    -   [Middlewares Interface](#middlewares-interface)
    -   [MySQL Interface](#mysql-interface)
    -   [Logger Interface](#todo)
    -   [Pagination Interface](#todo)

## Installation

To use the Uker package in your Go project, you can simply import it:

```go
import "github.com/unknowns24/uker"
```

Then, run `go get` to fetch the package:

```bash
go get github.com/unknowns24/uker
```

## Usage

Import the package and start using the utility functions and interfaces in your Go code. Below, you'll find details about the available interfaces and their functions.

## Interfaces

### HTTP Interface

The HTTP interface provides utilities for working with HTTP requests and responses, including pagination and body parsing.

#### Paginate

Paginate results from a database query.

```go
func Paginate(c *fiber.Ctx, db *gorm.DB, tableName string, condition string, result interface{}) (fiber.Map, error)
```

-   `c *fiber.Ctx`: The current Fiber context.
-   `db *gorm.DB`: The database connection.
-   `tableName string`: The name of the table to paginate.
-   `condition string`: The condition for filtering results.
-   `result interface{}`: A pointer to the variable where the paginated results will be stored.
-   Returns `fiber.Map`: A map containing pagination information, and an error if one occurs.

#### EndOutPut

Generate an HTTP response with a specified status code and message.

```go
func EndOutPut(c *fiber.Ctx, resCode int, message string, extraValues map[string]string) error
```

-   `c *fiber.Ctx`: The current Fiber context.
-   `resCode int`: The HTTP response status code.
-   `message string`: The message to include in the response.
-   `extraValues map[string]string`: Additional key-value pairs to include in the response.
-   Returns `error`: An error, if one occurs during response generation.

#### BodyParser

Parse the body of an HTTP request.

```go
func BodyParser(c *fiber.Ctx, requestInterface *interface{}) error
```

-   `c *fiber.Ctx`: The current Fiber context.
-   `requestInterface *interface{}`: A pointer to the variable where the parsed request body will be stored.
-   Returns `error`: An error if one occurs during body parsing.

### Middlewares Interface

The Middlewares interface provides utilities for working with middleware functions, including JWT token generation and authentication.

#### GenerateJWT

Generate a JSON Web Token (JWT).

```go
func GenerateJWT(id uint, keeplogin bool) (string, error)
```

-   `id uint`: The user ID for whom the JWT is generated.
-   `keeplogin bool`: Indicates whether to create a long-lived token.
-   Returns `(string, error)`: The generated JWT and an error if one occurs.

#### IsAuthenticated

Middleware function to check if a user is authenticated.

```go
func IsAuthenticated(c *fiber.Ctx) error
```

-   `c *fiber.Ctx`: The current Fiber context.
-   Returns `error`: An error if the user is not authenticated.

### MySQL Interface

The MySQL interface provides utilities for establishing a connection to a MySQL database.

#### StablishConnection

Establish a connection to a MySQL database.

```go
func StablishConnection(conn MySQLConnData, migrate ...interface{}) (db *gorm.DB, err error)
```

-   `conn MySQLConnData`: Connection data for the MySQL database.
-   `migrate ...interface{}`: Optional list of structures to perform database migrations.
-   Returns `(*gorm.DB, error)`: The database connection and an error if one occurs.
