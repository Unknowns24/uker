# Consumo de paginación por cursores desde el frontend

El paquete [`uker/pagination`](../uker/pagination) implementa paginación basada en
cursores firmados. Este documento describe qué parámetros debe enviar el
frontend, cómo se serializan filtros y ordenamientos, y qué funciones del
paquete debe invocar el backend para completar el flujo.

## Flujo general

1. **Frontend** construye la URL con `limit`, `sort`, filtros y (cuando exista)
   el cursor devuelto por la solicitud anterior.
2. **Handler** en el backend llama a `pagination.ParseWithSecurity` para
   validar la querystring y reconstruir parámetros a partir del cursor.
3. **Repositorio** aplica los parámetros con `pagination.Apply` para obtener
   los registros ordenados y filtrados desde la base de datos.
4. **Respuesta HTTP** se arma con `pagination.BuildPageSigned`, que genera
   `next_cursor` y `prev_cursor` usando los valores del último elemento de la
   página.

## Parámetros soportados en la querystring

| Parámetro        | Ejemplo                             | Descripción |
|------------------|-------------------------------------|-------------|
| `limit`          | `limit=25`                          | Tamaño de página (el paquete impone máximos configurables).
| `sort`           | `sort=created_at:desc,name:asc`     | Lista separada por comas. Cada entrada usa `campo[:asc\|desc]`. Si `id` no se indica, el paquete lo agrega para desempates.
| `cursor`         | `cursor=eyJ...`                     | Cursor firmado devuelto previamente. Debe reenviarse sin cambios.
| `<campo>_<op>`   | `status_in=active,pending`          | Filtros; `<op>` debe ser uno de `eq`, `neq`, `lt`, `lte`, `gt`, `gte`, `like`, `in`, `nin`.

> **Importante:** cuando la petición incluye `cursor`, no se pueden añadir ni
> modificar filtros u ordenamientos. `ParseWithSecurity` comparará la firma y
> rechazará la petición con `ErrInvalidSort` o `ErrInvalidFilter` si detecta
> inconsistencias.

### Campos con guiones bajos

`parseFilters` divide el nombre del campo y el operador utilizando el último
subrayado (`_`). Esto permite filtros como `document_number_like=46`, donde el
campo `document_number` conserva el sufijo del operador `like`.

## Construcción de la URL en el frontend

- **Primera página:** define explícitamente `limit`, `sort` (si se requiere un
  orden específico) y los filtros necesarios. Ejemplo genérico:

  ```http
  GET /recurso?limit=20&sort=created_at:desc&status_eq=active&name_like=juan
  ```

- **Siguientes páginas:** reutiliza el valor de `paging.next_cursor` o
  `paging.prev_cursor` retornado por el backend. No vuelvas a enviar filtros ni
  ordenamientos cuando se use un cursor; la firma ya codifica esos parámetros.

  ```http
  GET /recurso?cursor=eyJvZm...
  ```

El frontend no necesita conocer cómo se calcula el cursor: solo debe enviar los
parámetros permitidos y reenviar los cursores tal como fueron recibidos.

## Integración en el backend

Los proyectos que usen el paquete deben invocar las mismas funciones, aunque el
naming de handlers o repositorios varíe. El flujo recomendado es:

```go
params, err := pagination.ParseWithSecurity(r.URL.Query(), secret, ttl)
if err != nil {
    // responder con error 400/401 según corresponda
}

query := db.Model(&YourModel{})
query, err = pagination.Apply(query, params)
if err != nil {
    // responder con error 400 u otro según el caso
}

var results []YourModel
if err := query.Find(&results).Error; err != nil {
    // manejar error de base de datos
}

page, err := pagination.BuildPageSigned(params, results, params.Limit, extractor, secret)
if err != nil {
    // manejar error al generar cursores
}
```

- `ParseWithSecurity` valida `limit`, `sort`, filtros y, si llega un cursor,
  verifica la firma antes de reconstruir los parámetros originales.
- `Apply` traduce `params.Filters` y `params.Sort` en cláusulas SQL y solicita
  `limit+1` registros para calcular `has_more`.
- `BuildPageSigned` genera la estructura `pagination.PagingResponse`, truncando
  los resultados si exceden `limit` y calculando `next_cursor`/`prev_cursor`
  con los valores provistos por la función `extractor`.

La función `extractor` recibe cada elemento de la página y debe devolver un
mapa `map[string]string` con los campos que participaron en `params.Sort`. Un
extractor genérico puede usar reflexión para leer esos campos desde la
estructura de dominio.

## Estructura de la respuesta

El paquete retorna una estructura JSON consistente. Un ejemplo de respuesta es:

```json
{
  "data": [ /* resultados */ ],
  "paging": {
    "limit": 20,
    "has_more": true,
    "next_cursor": "eyJvZm...",
    "prev_cursor": ""
  }
}
```

Mientras el frontend respete el cursor firmado y mantenga los filtros iniciales,
podrá avanzar o retroceder páginas sin reimplementar la lógica de orden y
filtrado.
