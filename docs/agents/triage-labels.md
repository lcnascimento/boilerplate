# Triage Labels

The skills speak in terms of five canonical triage roles. This file maps those roles to the actual label strings used in this Project's issue tracker: the children of the single-select `triage` label group in Linear.

| Label in mattpocock/skills | Label in our tracker | Meaning                                  |
| -------------------------- | -------------------- | ---------------------------------------- |
| `needs-triage`             | `needs-triage`       | Maintainer needs to evaluate this issue  |
| `needs-info`               | `needs-info`         | Waiting on reporter for more information |
| `ready-for-agent`          | `ready-for-agent`    | Fully specified, ready for an AFK agent  |
| `ready-for-human`          | `ready-for-human`    | Requires human implementation            |
| `wontfix`                  | `wontfix`            | Will not be actioned                     |

The two category roles map to children of the single-select `type` label group:

| Category in mattpocock/skills | Label in our tracker |
| ----------------------------- | -------------------- |
| `bug`                         | `bug`                |
| `enhancement`                 | `feature`            |

When a skill mentions a role (e.g. "apply the AFK-ready triage label"), use the corresponding label string from these tables.
