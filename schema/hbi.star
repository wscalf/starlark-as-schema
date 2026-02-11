load("rbac.star", "v1_based_permission")
load("kessel.star", "relation", "assignable", "cardinality", "subref") #TODO: can we make these builtins or something?

host = {
    "workspace": relation(assignable("rbac", "workspace", cardinality.ExactlyOne)),

    "view": relation(subref("workspace", v1_based_permission("inventory", "host", "read", "inventory_host_view"))),
    "update": relation(subref("workspace", v1_based_permission("inventory", "host", "write", "inventory_host_update")))
}