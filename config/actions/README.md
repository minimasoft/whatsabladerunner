# Custom Actions

Custom Actions allow you to extend the bot's capabilities by defining HTTP requests that the LLM can trigger. These are defined as JSON files in `config/actions/*.json`.

## Configuration Schema

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique name for the action (used in the LLM protocol). |
| `description` | string | Description explaining when and how to use the action. |
| `method` | string | HTTP method (`GET`, `POST`, `PUT`, `DELETE`). Defaults to `POST`. |
| `url` | string | The endpoint URL. Supports variable templating. |
| `response_to_llm` | boolean | If `true`, the HTTP response body is sent back to the LLM. |
| `parameters` | object | JSON Schema for the parameters the LLM should provide. |

## URL Templating

You can use placeholders in the `url` field like `{parameter_name}`. When the action is executed, these placeholders will be replaced by the values provided by the LLM.

- **Path Parameters**: `http://api.com/users/{id}`
- **Query Parameters (Explicit)**: `http://api.com/search?query={term}`

If a parameter is NOT used as a template in the URL:
- For `POST`, `PUT`, or `PATCH`: It will be included in the JSON request body.
- For `GET` or `DELETE`: It will be automatically appended as a query parameter (e.g., `?param=value`).

## Examples

### 1. Simple POST (Save to Calendar)
```json
{
    "name": "save_to_calendar",
    "description": "Save an event to the calendar.",
    "method": "POST",
    "url": "https://api.example.com/events",
    "response_to_llm": true,
    "parameters": {
        "type": "object",
        "properties": {
            "title": { "type": "string" },
            "date": { "type": "string" }
        },
        "required": ["title", "date"]
    }
}
```

### 2. GET with Path Parameter (Get User Info)
```json
{
    "name": "get_user_info",
    "description": "Retrieve information about a user by ID.",
    "method": "GET",
    "url": "https://api.example.com/users/{user_id}",
    "response_to_llm": true,
    "parameters": {
        "type": "object",
        "properties": {
            "user_id": { "type": "string" }
        },
        "required": ["user_id"]
    }
}
```

### 3. DELETE Action
```json
{
    "name": "delete_document",
    "description": "Delete a document by its reference.",
    "method": "DELETE",
    "url": "https://api.example.com/docs/{doc_id}",
    "parameters": {
        "type": "object",
        "properties": {
            "doc_id": { "type": "string" }
        },
        "required": ["doc_id"]
    }
}
```
