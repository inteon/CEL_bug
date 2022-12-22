# CEL_bug

This repo contains an example of the flake failure I see when creating a CR with CEL rules (using HTTP POST):
```
Invalid value: "object": internal error: runtime error: index out of range [3] with length 3 evaluating rule: <rule name>
```
this is the CEL rule:
`(has(self.prop1) ? 1 : 0) + (has(self.prop2) ? 1 : 0) == 1`

The sent JSON is valid and works most of the time.

K8S version: v1.25.0
