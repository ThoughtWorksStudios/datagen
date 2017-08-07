{

  function rootNode(statements) {
    return {
      kind: "root",
      children: searchNodes(statements)
    };
  }

  function importNode(path) {
    return {
      kind: "import",
      value: path
    };
  }

  function genNode(entity, args) {
    return {
      kind: "generation",
      value: entity,
      args: args || []
    }
  }

  function assignNode(identNode) {
    return {
      kind: "assignment",
      name: identNode.value,
    }
  }

  function entityNode(assignment, entity) {
    if (assignment) {
      entity.name = assignment.name
    }
    return entity;
  }

  function entityDefNode(extended, body) {
    var node = {
      kind: "entity",
      children: body || []
    };

    if (extended) {
      node.related = extended
    }

    return node;
  }

  function staticFieldNode(ident, fieldValue) {
    return {
      kind: "field",
      name: ident.value,
      value: fieldValue
    };
  }

  function dynamicFieldNode(ident, fieldType, args, bound) {
    return {
      kind:  "field",
      name:  ident.value,
      value: fieldType,
      args:  args || [],
      bound: bound || []
    };
  }

  function idNode(name) {
    return {
      kind: "identifier",
      value: name
    };
  }

  function builtinNode(value) {
    return {
      kind: "builtin",
      value: value
    };
  }

  function dateLiteralNode(date, localTime) {
    if (!!localTime && !!localTime) {
      date += localTime;
    }

    return {
      kind:  "literal-date",
      value: new Date(date)
    };
  }

  function floatLiteralNode(s) {
    return {
      kind:  "literal-float",
      value: parseFloat(s)
    };
  }

  function intLiteralNode(s) {
    return {
      kind:  "literal-int",
      value: parseInt(s, 10)
    };
  }
  function boolLiteralNode(value) {
    return {
      kind:  "literal-bool",
      value: "true" === value.toLowerCase()
    };
  }

  function strLiteralNode(value) {
    return {
      kind:  "literal-string",
      value: value
    };
  }

  function nullLiteralNode() {
    return {
      kind: "literal-null",
      value: null
    }
  }

  function searchNodes(v) {
    if (!v || (Array.isArray(v) && !v.length)) return [];
    if (v && "string" === typeof v.kind) return [v];

    for (var i = 0, res = [], cur, len = v.length; i < len; ++i) {
      cur = v[i];

      if (cur && "string" === typeof cur.kind) {
        res.push(cur);
      } else {
        if (Array.isArray(cur)) {
          res = res.concat(searchNodes(cur));
        }
      }
    }

    return res;
  }

  // nabbed from jQuery :)
  function trim(s) {
    if (!s) return "";
    return (s + "").replace(/^[\s\uFEFF\xA0]+|[\s\uFEFF\xA0]+$/g, "");
  }

  function delimitedNodeSlice(first, rest) {
    var res = [first];

    if (rest) {
      res = res.concat(searchNodes(rest));
    }

    return res;
  }
}

Script = prog:Statement* EOF {
  return rootNode(prog);
} / .* EOF { error(`Don't know how to evaluate:\n${text()}`); }

Statement = statement:(ImportStatement / GenerateExpr / EntityExpr / Comment) {
  return statement;
}

ImportStatement = _ "import" _ path:StringLiteral _ {
  var fspath = trim(path.value);
  if ("" === fspath) {
     error("import statement requires a resolvable path");
  } else {
    return importNode(fspath);
  }
} / FailOnBadImport

GenerateExpr = _ "generate" _ '(' _ count:SingleArgument _ ',' _ entity:EntityRef _ ')' _ {
  if ("literal-int" !== count.kind) {
     error("`generate` takes a non-zero integer count as its first argument");
  }

  return genNode(entity, [count]);
} / FailOnUnterminatedGeneratorArguments / FailOnMissingGenerateArguments

Assignment = name:Identifier _ ASSIGN_OP {
  if (!name) {
    return error("Missing left-hand assignment");
  }
  return assignNode(name);
}

EntityRef = EntityExpr / Identifier

EntityExpr "entity expression" = _ name:Assignment? _ entity:EntityDefinition _ {
  return entityNode(name, entity);
} / FailOnMissingRightHandAssignment

EntityDefinition = extended:Identifier? _ '{' _ body:FieldSet? _ '}' {
  return entityDefNode(extended, body);
} / FailOnUnterminatedEntity

FieldSet "entity fields" = FailOnUndelimitedFields / first:FieldDecl rest:(_ ',' _ FieldDecl)* (_ ',')? {
  if (!first) {
    return error("Missing field declaration");
  }

  return delimitedNodeSlice(first, rest);
}

FieldDecl = StaticDecl / DynamicDecl / FailOnMissingFieldType

StaticDecl "field declaration" = name:Identifier _ fieldValue:Literal _ {
  if (!name || !fieldValue) {
    return error("Field declaration requires both field name and field type");
  }

  return staticFieldNode(name, fieldValue);
}

DynamicDecl "field declaration" = name:Identifier _ fieldType:(Builtin / EntityRef) _ args:Arguments? _ bound:Bound? _ {
  if (!name || !fieldType) {
    return error("Field declaration requires both field name and field type");
  }

  return dynamicFieldNode(name, fieldType, args, bound);
}

Bound = '[' _ body:ArgumentsBody? _ ']' {
  return body || [];
} / FailOnUnterminatedBound

Arguments = '(' _ body:ArgumentsBody? _ ')' {
  return body || [];
} / FailOnUnterminatedArguments

ArgumentsBody "arguments body" = FailOnUndelimitedArgs / first:SingleArgument rest:(_ ',' _ SingleArgument)* {
  if (!first) {
    return error("Missing argument");
  }

  return delimitedNodeSlice(first, rest);
}

Literal = DateTimeLiteral / NumberLiteral / BoolLiteral / StringLiteral / NullLiteral

SingleArgument = Literal / Identifier

Identifier = !ReservedWord [a-z0-9_$]i+ {
  var val = text();

  if (val.indexOf("$") !== -1) {
     error(`Illegal identifier ${JSON.stringify(val)}; identifiers start with a letter or underscore, followed by zero or more letters, underscores, and numbers`);
  }

  if (/^[0-9]/.test(val)) {
     error(`Illegal identifier ${JSON.stringify(val)}; identifiers start with a letter or underscore, followed by zero or more letters, underscores, and numbers`);
  }

  return idNode(val);
} / FailOnIllegalIdentifier

Builtin "built-in types" = FieldTypes {
  return builtinNode(text());
}

DateTimeLiteral = date:IsoDate localTime:LocalTimePart? {
  return dateLiteralNode(date, localTime);
} / FailOnMissingDate

LocalTimePart = ts:TimePart zone:ZonePart? {
  if (!zone) return ts;
  return ts + zone;
}

IsoDate = DIGIT DIGIT DIGIT DIGIT '-' DIGIT DIGIT '-' DIGIT DIGIT { return text(); }
TimePart = 'T'i DIGIT DIGIT ':' DIGIT DIGIT ':' DIGIT DIGIT { return text().toUpperCase(); }
ZonePart = 'Z'i { return "Z"; } / [+-] DIGIT DIGIT ':'? DIGIT DIGIT { return text().replace(/:/g, ""); }

NumberLiteral = '-'? INT ('.' DIGIT+)? {
  var s = text();
  if (s.indexOf(".") !== -1) {
    return floatLiteralNode(s);
  } else {
    return intLiteralNode(s);
  }
} / FailOnOctal

BoolLiteral = BoolToken {
  return boolLiteralNode(text());
}

NullLiteral = NullToken {
  return nullLiteralNode();
}

StringLiteral = '"' chars:CHAR* '"' {
  return strLiteralNode(chars.join(""));
}

CHAR = NonescapedChar / EscapedChar

ESCAPE = "\\"

NonescapedChar = [^\0-\x1F\x22\x5C]

EscapedChar = ESCAPE sequence:(LITERAL_SEQ / INVISIBLE_SEQ / UNICODE_SEQ) { return sequence; }

UNICODE_SEQ = 'u' digits:(HEXDIG HEXDIG HEXDIG HEXDIG) {
  return String.fromCharCode(parseInt(digits.join(""), 16));
}

INVISIBLE_SEQ =
      'b' { return "\b"; }
      / 'f' { return "\f"; }
      / 'n' { return "\n"; }
      / 'r' { return "\r"; }
      / 't' { return "\t"; }

LITERAL_SEQ = '"' / '\\' / '/'

ASSIGN_OP = ':'

INT = '0' / NON_ZERO DIGIT*

NON_ZERO = [1-9]

DIGIT = [0-9]

HEXDIG = [0-9a-f]i

ReservedWord = Keyword / FieldTypes / NullToken / BoolToken

Keyword = "import" / "generate"

FieldTypes = "integer" / "decimal" / "string" / "date" / "dict"

NullToken = "null"

BoolToken = "true" / "false"

/**
 *  88 88b 88 Yb    dP    db    88     88 8888b.      88""Yb 88   88 88     888888 .dP"Y8
 *  88 88Yb88  Yb  dP    dPYb   88     88  8I  Yb     88__dP 88   88 88     88__   `Ybo."
 *  88 88 Y88   YbdP    dP__Yb  88  .o 88  8I  dY     88"Yb  Y8   8P 88  .o 88""   o.`Y8b
 *  88 88  Y8    YP    dP""""Yb 88ood8 88 8888Y"      88  Yb `YbodP' 88ood8 888888 8bodP'
 */

FailOnBadImport "invalid import statment" = "import" _ [^ \t\r\n]* { error("import statement requires a path"); }
FailOnOctal "octal numbers not supported" = "\\0" DIGIT+ { error("Octal sequences are not supported"); };
FailOnUnterminatedEntity "unterminated entity" = _ Identifier? _ '{' _ FieldSet? _ EOF { error("Unterminated entity expression (missing closing curly brace"); }
FailOnUndelimitedFields "missing field delimiter" = FieldDecl (_ "," _) (_ "," _)+ { error("Expected another field declaration"); } / FieldDecl (_ FieldDecl)+ { error("Multiple field declarations must be delimited with a comma"); }
FailOnUnterminatedBound "unterminated bound" = '[' _ ArgumentsBody? _ (!SingleArgument [^)] / EOF) { error("Unterminated bound list (missing closing square bracket)"); }
FailOnUnterminatedArguments "unterminated arguments" = '(' _ ArgumentsBody? _ (!SingleArgument [^)] / EOF) { error("Unterminated argument list (missing closing parenthesis)"); }
FailOnUndelimitedArgs "missing argument delimiter" = SingleArgument ((_ / _ [^,})] _) SingleArgument)+ { error("Multiple arguments must be delimited with a comma"); }
FailOnIllegalIdentifier "illegal identifier" = ReservedWord { error(`Illegal identifier: ${JSON.stringify(text())} is a reserved word`); }
FailOnMissingDate "timestamps must have date" = LocalTimePart { error("Must include ISO-8601 (YYYY-MM-DD) date as part of timestamp"); };
FailOnMissingGenerateArguments = _ "generate" _ (EntityRef / '(' _ (EntityRef / SingleArgument) _ ')') _ { error(`\`generate\` statement ${JSON.stringify(text())} requires arguments \`(count, entity)\``); }
FailOnUnterminatedGeneratorArguments = _ "generate" _ '(' _ ((EntityRef / SingleArgument) (_ ',' _ (EntityRef / SingleArgument))*)? _ [^)] _ { error(`\`generate\` statement ${JSON.stringify(text())} requires arguments \`(count, enitty)\``); }
FailOnMissingFieldType = Identifier { error(`Missing field type for field declaration ${JSON.stringify(text())}`); }
FailOnMissingRightHandAssignment = ass:Assignment {
  if (!ass) { // hehe, I said "ass".
    error(`Bad identifier ${JSON.stringify(text())}`);
  }
   error(`Missing right-hand of assignment expression ${JSON.stringify(text())}`);
}

/**
 *  888888 88b 88 8888b.
 *  88__   88Yb88  8I  Yb
 *  88""   88 Y88  8I  dY
 *  888888 88  Y8 8888Y"
 */

Comment = '#' (!EOL .)* EOL

BLANK "whitespace" = [ \t\r\n]

_ "ignored" = (BLANK / Comment)*

EOL = [\n\r]

EOF = !.
