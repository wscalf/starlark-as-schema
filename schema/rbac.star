load("kessel.star", "relation", "assignable", "cardinality", "union", "intersect", "subref", "ref", "all", "uuid")

# Everything in this file has reporter = rbac

principal = {}

role = {
    "any_any_any": relation(assignable("rbac", "principal", cardinality.All, all("rbac", "principal")))
}

role_binding = {
    "granted": relation(assignable("rbac", "role", cardinality.Any, uuid())),
    "subject": relation(assignable("rbac", "principal", cardinality.Any, uuid()))
}

workspace = {
    "binding": relation(assignable("rbac", "role_binding", cardinality.Any, uuid())),
    "parent": relation(assignable("rbac", "workspace", cardinality.ExactlyOne, uuid()))
}

def v1_based_permission(application, resource, verb, v2_perm):
    # Role permissions
    app_admin = _add_v1_role_permission("{}_any_any".format(application))
    any_verb = _add_v1_role_permission("{}_{}_any".format(application, resource))
    any_resource = _add_v1_role_permission("{}_any_{}".format(application, verb))
    v1_perm = _add_v1_role_permission("{}_{}_{}".format(application, resource, verb))


    # add_member is a native method provided by the interpreter that adds the given relation to the type by name
    # this is necessary because starlark heap memory is frozen after load, so code in a module isn't allowed to modify the contents of that module when called by others
    add_member("rbac", "role", v2_perm, relation(union(ref("any_any_any"), union(ref(app_admin), union(ref(any_verb), union(ref(any_resource), ref(v1_perm)))))))

    # Role binding permission
    add_member("rbac", "role_binding", v2_perm, relation(intersect(ref("subject"), subref("granted", v2_perm))))

    # Workspace permission
    add_member("rbac", "workspace", v2_perm, relation(union(subref("binding", v2_perm), subref("parent", v2_perm))))

    return v2_perm

def _add_v1_role_permission(name):
    add_member("rbac", "role", name, relation(assignable("rbac", "principal", cardinality.All, all("rbac", "principal"))))
    return name
