#!/usr/bin/env python3
"""
Generate a .env file by reading the entries from env.schema.json

TODO !!! READ, REVIEW, AND FIX
"""

import json
import time
from collections.abc import Mapping
from pathlib import Path
from typing import Any, NotRequired, TypedDict, cast

from scripts.env_vars import (
    EnvVar,
    EnvVarsRoot,
    EnvVarsSection,
    TypeHandlerRegistry,
    get_value_for_type,
)
from scripts.printer import Printer
from scripts.validator import Validator

# Resolve paths relative to the repository root (parent of scripts/)
_REPO_ROOT = Path(__file__).resolve().parents[1]
SCHEMA_PATH = _REPO_ROOT / "scripts" / "env.schema.json"
OUTPUT_ENV_PATH = _REPO_ROOT / f".env.generated.{int(time.time())}"

class SchemaVariable(TypedDict):
    name: str
    type: str
    description: str
    value: NotRequired[str]

class SchemaSection(TypedDict):
    name: str
    description: str
    variables: list[SchemaVariable]

class SchemaRoot(TypedDict):
    prefix: str
    sections: list[SchemaSection]

def load_schema_raw(path: Path) -> SchemaRoot:
    if not path.exists():
        raise ValueError(f"Schema file not found at {path.resolve()}")
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


class SchemaValidator:
    """
    Converts the JSON schema into a normalized structure (RawSection/RawVar).

    Supported formats:
    - Legacy (current):
        {
          "GENERAL": {
            "description": "...",
            "variables": {
              "SERVER_IP": {"type": "IP", "description": "...", "value": null},
              ...
            }
          },
          "ADGUARD": { ... }
        }

    - Proposed (easier):
        {
          "prefix": "HOMELAB",
          "sections": [
            {
              "name": "GENERAL",
              "description": "...",
              "variables": [
                {"name": "SERVER_IP", "type": "IP", "description": "...", "value": null},
                ...
              ]
            },
            ...
          ]
        }
    """

    def __init__(self, raw: dict[str, Any]) -> None:
        self.raw = raw

    def parse(self) -> None:
        """
        Returns (prefix_override, sections)
        """
        self._parse_proposed(self.raw)
        raise ValueError("Unsupported schema root. Expecting an object or an object with 'sections'.")

    def _parse_proposed(self, raw_root: object) -> None:
        if not isinstance(raw_root, Mapping):
            raise ValueError("Schema root must be an object.")
        root = cast(Mapping[str, object], raw_root)
        if "prefix" not in root or "sections" not in root:
            raise ValueError("Schema root must contain 'prefix' and 'sections'.")

        prefix = root["prefix"]
        sections = root["sections"]

        prefix = raw_root.prefix
        Validator.validate_string(prefix, "prefix")

        if not isinstance(raw_root.get("sections"), list):
            raise ValueError("'sections' must be an array.")

        raw_sections: list[dict[str, Any]] = raw_root.get("sections", [])
        if not raw_sections:
            raise ValueError("'sections' must not be empty.")

        for idx, raw_section_obj in enumerate(raw_sections, start=1):
            if not isinstance(raw_section_obj, dict):
                raise ValueError(f"Section at index {idx} must be an object.")
            section_name = raw_section_obj.get("name")
            section_desc = raw_section_obj.get("description")
            section_variables = raw_section_obj.get("variables", [])

            Validator.validate_string(section_name, f"Section[{idx}].name")
            Validator.validate_string(section_desc, f"Section[{idx}].description")
            if not isinstance(section_variables, list):
                raise ValueError(f"Section[{idx}].variables must be an array.")

            raw_vars: list[RawVar] = []
            for jdx, var_obj in enumerate(section_variables, start=1):
                if not isinstance(var_obj, dict):
                    raise ValueError(f"Section[{idx}].variables[{jdx}] must be an object.")
                var_name = var_obj.get("name")
                var_type = var_obj.get("type")
                var_desc = var_obj.get("description")
                var_value = var_obj.get("value")

                Validator.validate_string(var_name, f"Section[{idx}].variables[{jdx}].name")
                Validator.validate_string(var_type, f"Section[{idx}].variables[{jdx}].type")
                Validator.validate_string_or_none(var_desc, f"Section[{idx}].variables[{jdx}].description")
                Validator.validate_string_or_none(var_value, f"Section[{idx}].variables[{jdx}].value")

                raw_vars.append(RawVar(name=var_name, type=var_type, description=var_desc, value=var_value))

            sections.append(RawSection(name=section_name, description=section_desc, variables=raw_vars))

        return prefix, sections

    def _validate_section(self, section: dict[str, Any]) -> None:
        pass


# -----------------------------
# Builder: dotenv content
# -----------------------------


class DotenvBuilder:
    def __init__(self) -> None:
        self._lines: list[str] = []
        self.total_vars = 0

    def add_section(self, section: EnvVarsSection) -> None:
        self._lines.append("#" * 120)
        self._lines.append(f"# {section.name}")
        if section.description:
            self._lines.append(f"# {section.description}")
        self._lines.append("#" * 120)

        for var in section.vars:
            self._lines.append(var.get_dotenv_value())
            self.total_vars += 1

        self._lines.append("")

    def build(self) -> str:
        return "\n".join(self._lines).rstrip() + "\n"


# -----------------------------
# Orchestration
# -----------------------------


def parse_schema(obj: Any, *, registry: TypeHandlerRegistry | None = None) -> EnvVarsRoot:
    adapter = SchemaValidator(obj)
    parsed = adapter.parse()
    if isinstance(parsed, tuple):
        prefix_override, sections = parsed
    else:
        # Defensive: shouldn't happen due to return types above
        prefix_override, sections = None, parsed

    # TODO extract prefix from JSON
    root = EnvVarsRoot()
    if prefix_override:
        root.prefix = prefix_override

    for section_data in sections:
        section_full_name = f"{root.prefix}_{section_data.name}"
        section = EnvVarsSection(name=section_full_name, description=section_data.description)
        root.add_section(section)

        # Print section header for user context
        Printer.info(f"\n\n>>>>>>>>>> Section: {section.name}")
        if section.description:
            for line in Printer.wrap_lines(text=section.description, width=120):
                Printer.info(line)

        for raw_var in section_data.variables:
            var = EnvVar(
                name=f"{section.name}_{raw_var.name}",
                type=raw_var.type,
                description=raw_var.description,
                value=raw_var.value,
            )
            section.add_var(var)

            value = get_value_for_type(
                var_name=var.name,
                var_description=var.description,
                var_type=var.type,
                var_value=var.value,
                registry=registry,
            )
            var.set_value(value)

    return root


def main() -> None:
    raw = load_schema_raw(SCHEMA_PATH)
    schema: EnvVarsRoot = parse_schema(raw)

    builder = DotenvBuilder()
    for section in schema.sections:
        builder.add_section(section)

    content = builder.build()
    OUTPUT_ENV_PATH.write_text(content, encoding="utf-8")
    Printer.info(f"Wrote {OUTPUT_ENV_PATH} with {builder.total_vars} entries.")


if __name__ == "__main__":
    main()
