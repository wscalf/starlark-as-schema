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

def assignable(namespace, type, cardinality, data_type):
    return struct(kind = "assignable", namespace = namespace, type = type, cardinality = cardinality, data_type=data_type)
    
def field(type, required=False):
    return struct(kind = "field", required = required, type = type)

def text(minLength=None, maxLength=None, regex=None):
    return struct(kind="text", minLength=minLength, maxLength=maxLength, regex=regex)

def numeric_id(min=None, max=None):
    return struct(kind="numeric_id", min=min, max=max)

def uuid():
    return struct(kind="uuid")
    
def all(namespace, name):
    return text(regex="{}/{}:\\*".format(namespace, name))