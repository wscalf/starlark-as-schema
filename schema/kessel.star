cardinality = struct(
    Any = "Any",
    AtMostOne = "AtMostOne",
    ExactlyOne = "ExactlyOne",
    AtLeastOne = "AtLeastOne",
    All = "All"
)

def relation(body):
    return struct(kind = "relation", body = body)

def intersect(left, right):
    return struct(kind = "and", left = left, right = right)

def union(left, right):
    return struct(kind = "or", left = left, right = right)

def exclude(left, right):
    return struct(kind = "unless", left = left, right = right)

def ref(name):
    return struct(kind = "ref", name = name)

def subref(name, subname):
    return struct(kind = "subref", name = name, subname = subname)

def assignable(namespace, type, cardinality):
    return struct(kind = "assignable", namespace = namespace, type = type, cardinality = cardinality)
    
def field():
    return struct(kind = "field")