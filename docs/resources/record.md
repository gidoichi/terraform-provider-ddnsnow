---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "ddnsnow_record Resource - ddnsnow"
subcategory: ""
description: |-
  
---

# ddnsnow_record (Resource)



## Example Usage

```terraform
resource "ddnsnow_record" "a_record" {
  type  = "A"
  value = "127.0.0.1"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `type` (String) The record type. One of: `A`, `AAAA`, `CNAME`, `TXT`, `NS`.
- `value` (String) The record value.
