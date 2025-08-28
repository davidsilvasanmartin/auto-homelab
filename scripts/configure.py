#!/usr/bin/env python3
"""
Generate a .env file by reading the entries from env.schema.json

TODO !!! READ, REVIEW, AND FIX
"""

import json
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

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


def load_schema_raw(path: Path) -> dict | list:
    if not path.exists():
        raise ValueError(f"Schema file not found at {path.resolve()}")
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


# -----------------------------
# Adapter: support multiple schema shapes
# -----------------------------


@dataclass
class RawVar:
    name: str
    type: str
    description: str | None
    value: str | None


@dataclass
class RawSection:
    name: str
    description: str | None
    variables: list[RawVar]


class SchemaAdapter:
    """
    Converts different JSON schema shapes into a normalized structure (RawSection/RawVar).

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

    def __init__(self, raw: dict | list) -> None:
        self.raw = raw

    def parse(self) -> tuple[str | None, list[RawSection]]:
        """
        Returns (prefix_override, sections)
        """
        if isinstance(self.raw, dict) and "sections" in self.raw and isinstance(self.raw["sections"], list):
            return self._parse_proposed(self.raw)
        if isinstance(self.raw, dict):
            return None, self._parse_legacy(self.raw)
        raise ValueError("Unsupported schema root. Expecting an object or an object with 'sections'.")

    def _parse_legacy(self, obj: dict) -> list[RawSection]:
        Validator.validate_dict(value=obj, name="Schema root")
        sections: list[RawSection] = []
        for section_name, section_val in obj.items():
            Validator.validate_string(value=section_name, name="Section name")
            Validator.validate_dict(value=section_val, name=f"Section '{section_name}'")

            section_description = section_val.get("description")
            section_vars = section_val.get("variables")
            Validator.validate_string(value=section_description, name=f"Section '{section_name}' description")
            Validator.validate_dict(value=section_vars, name=f"Section '{section_name}' variables")

            raw_vars: list[RawVar] = []
            for var_name, var_obj in section_vars.items():
                Validator.validate_string(
                    value=var_name, name=f"Section '{section_name}' variable '{var_name}"
                )
                Validator.validate_dict(value=var_obj, name=f"Section '{section_name}' variable '{var_name}'")

                var_type = var_obj.get("type")
                var_description = var_obj.get("description")
                var_value = var_obj.get("value")

                Validator.validate_string(
                    value=var_type, name=f"Section '{section_name}' variable '{var_name}' type"
                )
                Validator.validate_string_or_none(
                    value=var_description, name=f"Section '{section_name}' variable '{var_name}' description"
                )
                Validator.validate_string_or_none(
                    value=var_value, name=f"Section '{section_name}' variable '{var_name}' default value"
                )

                raw_vars.append(
                    RawVar(name=var_name, type=var_type, description=var_description, value=var_value)
                )

            sections.append(
                RawSection(name=section_name, description=section_description, variables=raw_vars)
            )
        return sections

    def _parse_proposed(self, obj: dict) -> tuple[str | None, list[RawSection]]:
        # prefix is optional override
        prefix = obj.get("prefix")
        if prefix is not None:
            Validator.validate_string(prefix, "prefix")

        sections_node = obj.get("sections", [])
        if not isinstance(sections_node, list):
            raise ValueError("'sections' must be an array.")

        sections: list[RawSection] = []
        for idx, section_obj in enumerate(sections_node, start=1):
            if not isinstance(section_obj, dict):
                raise ValueError(f"Section at index {idx} must be an object.")
            section_name = section_obj.get("name")
            section_desc = section_obj.get("description")
            variables_node = section_obj.get("variables", [])

            Validator.validate_string(section_name, f"Section[{idx}].name")
            Validator.validate_string(section_desc, f"Section[{idx}].description")
            if not isinstance(variables_node, list):
                raise ValueError(f"Section[{idx}].variables must be an array.")

            raw_vars: list[RawVar] = []
            for jdx, var_obj in enumerate(variables_node, start=1):
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
    adapter = SchemaAdapter(obj)
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
