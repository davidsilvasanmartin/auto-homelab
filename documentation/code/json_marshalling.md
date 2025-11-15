### Nuances of JSON Marshalling
The following JSON
```json
{
  "variables": ["a","b"]
}
```
WILL marshall into the following configuration with no errors
```go
type Root struct {
	// should be `json:"variables"`  !!!
	Variables   []string `json:"vars"`
}
```
producing an object such as the following one:
```go
obj := &Root{
	Variables: []string{"a", "b"}
}
```
EVEN IF THE KEY `vars` in the JSON configuration of the `type Root` is incorrect!!
What happens is that Go sees the `variables` keys and transforms it into the `Variables`
property of the `Root` struct, even if `json:"vars"` is specified,
as opposed to `json:"variables"`.