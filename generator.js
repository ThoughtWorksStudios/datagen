;(function() {
  "use strict";

  const _ = require("lodash");
  const uuid = require("hyperid")();
  const faker = require("faker");
  const dateformat = require("dateformat");

  function Generator(name, parent) {
    this.name = name;
    this.fields = {};
    this.fields["$id"] = new UuidField();

    if (parent) {
      this.base = parent.type();

      this.fields["$type"] = new LiteralField({value: this.type()});
      this.fields["$species"] = new LiteralField({value: this.name});
      this.fields["$extends"] = new LiteralField({value: this.base});

      _.each(Object.keys(parent.fields), (key) => {
        if (!/^\$/.test(key)) {
          this.fields[key] = new ReferenceField({key: key, generator: parent});
        }
      });
    }
  }

  Generator.prototype.type = function type() {
    if ((this.name.startsWith("$") || "" === this.name) && "" !== this.base) {
      return this.base;
    }

    return this.name;
  }

  Generator.prototype.generate = function generate(count) {
    if (count === 1) return this.one();

    var result = new Array(count);

    for (var i = 0; i < count; ++i) {
      result[i] = this.one();
    }

    return result;
  }

  Generator.prototype.one = function generateSingle() {
    return _.reduce(this.fields, (result, field, name) => {
      result[name] = field.value();
      return result;
    }, {});
  }

  Generator.prototype.withField = function resolveField(name, fieldType, options) {
    var mkField;

    switch (fieldType) {
      case "string":
        mkField = StringField;
        break;
      case "integer":
        mkField = IntegerField;
        break;
      case "decimal":
        mkField = FloatField;
        break;
      case "date":
        mkField = DateField;
        break;
      case "bool":
        mkField = BoolField;
        break;
      case "dict":
        mkField = DictField;
        break;
      case "entity":
        mkField = EntityField;
        break;
      case "literal":
        mkField = LiteralField;
        break;
      default:
        throw new Error(`Don't know how to handle ${fieldType}`);
    }

    this.fields[name] = new mkField(options);
    return this;
  };

  function Field(config) {
    this.one = function missingImpl() { throw new Error("one() must be implemented by subclasses"); }
    this.value = function generateValue() {
      if (!config || !config.countRange) {
        return this.one();
      }
      var result = [];
      for (var i = 0, count = config.countRange.count(); i < count; ++i) {
        result.push(this.one());
      }
      return result;
    }
  }

  function ReferenceField(config) {
    Field.call(this, config);
    this.one = function resolveValueFromParent() { return config.generator.fields[config.key].value(); };
  }

  function EntityField(config) {
    Field.call(this, config);
    this.one = function makeNested() {
      return config.entity.generate(1); // stub count to be always 1 for now
    };
  }

  function UuidField() {
    Field.call(this);
    this.one = uuid
  }

  function BoolField(config) {
    Field.call(this, config);
    this.one = function randBool() {
      return Math.random() > 0.49;
    };
  }

  function LiteralField(config) {
    Field.call(this, config);
    this.one = function constantVal() {
      return config.value;
    };
  }

  function StringField(config) {
    Field.call(this, config);
    this.one = function randString() {
      return faker.random.alphaNumeric(config.len);
    };
  }

  function IntegerField(config) {
    Field.call(this, config);
    this.one = function randInt() {
      return faker.random.number(config);
    };
  }

  function FloatField(config) {
    Field.call(this, config);

    this.one = function randFloat() {
      return parseFloat(faker.finance.amount(config.min, config.max, config.precision));
    };
  }

  function DateField(config) {
    Field.call(this, config);
    var min = dateformat(config.min, "isoUtcDateTime"),
      max = dateformat(config.max, "isoUtcDateTime");

    this.one = function stubString() {
      return faker.date.between(min, max);
    };
  }

  function DictField(config) {
    Field.call(this, config);
    this.one = function stubString() {
      return `from dictionary ${config.name}`;
    };
  }

  module.exports = {
    Generator
  };
})();
