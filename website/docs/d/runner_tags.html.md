---
layout: "rundeck"
page_title: "Rundeck: rundeck_runner_tags"
sidebar_current: "docs-rundeck-datasource-runner-tags"
description: |-
  The rundeck_runner_tags data source discovers runner tags in use and their usage counts.
---

# rundeck\_runner\_tags

Discovers runner tags currently in use for a project and how many runners use each one. Useful for validating tag names before assigning them consistently across a large runner set.

**Requirements:** Requires Rundeck Enterprise 5.17.0+ (API v56). Configure the provider with `api_version = "56"` or higher. The underlying Rundeck API (`ListProjectAssociatedTags`) requires a project scope — there is no system-wide mode for this endpoint, despite what its name might suggest.

## Example Usage

```hcl
data "rundeck_runner_tags" "project_scoped" {
  project_name = "my-project"
}

output "known_tags" {
  value = keys(data.rundeck_runner_tags.project_scoped.tags)
}
```

## Argument Reference

* `project_name` - (Required) Scope tag discovery to runners associated with this project.

## Attributes Reference

* `tags` - Map of tag name to the number of runners using that tag.

## Notes

This data source deliberately does not expose keyword search or pagination (the underlying `SearchTags` API) — it's intended for tag discovery/validation, not autocomplete. It also does not duplicate `data.rundeck_runner`'s `tag_names` attribute, which already covers listing tags for a single, specific runner.
