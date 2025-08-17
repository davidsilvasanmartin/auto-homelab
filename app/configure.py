#!/usr/bin/env python3
"""
Generate a .env file by reading the entries from env.schema.json
"""
import json
from pathlib import Path
from typing import Any
import time

from app.printer import Printer
from app.validator import Validator
from app.env_vars import EnvVarsRoot, EnvVarsSection, EnvVar, get_value_for_type

# Resolve paths relative to the repository root (parent of app/)
_REPO_ROOT = Path(__file__).resolve().parents[1]
SCHEMA_PATH = _REPO_ROOT / "app" / "env.schema.json"
OUTPUT_ENV_PATH = _REPO_ROOT / f".env.generated.{int(time.time())}"

def load_schema_raw(path: Path) -> dict:
    if not path.exists():
        raise ValueError(f"Schema file not found at {path.resolve()}")
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)

def parse_schema(obj: Any) -> EnvVarsRoot:
    Validator.validate_dict(value=obj, name="Schema root")

    root = EnvVarsRoot()

    for section_name, section_val in obj.items():
        Validator.validate_string(value=section_name, name="Section name")
        Validator.validate_dict(value=section_val, name=f"Section '{section_name}'")

        section_description = section_val.get("description")
        section_vars = section_val.get("variables")
        Validator.validate_string(value=section_description, name=f"Section '{section_name}' description")
        Validator.validate_dict(value=section_vars, name=f"Section '{section_name}' variables")

        section: EnvVarsSection = EnvVarsSection(name=f"{root.prefix}_{section_name}", description=section_description)
        root.add_section(section)

        # Print the current section header for user context
        Printer.info(f"\n\n>>>>>>>>>> Section: {section.name}")
        if section.description:
            for line in Printer.wrap_lines(text=section.description, width=120):
                Printer.info(line)

        for var_name, var_obj in section_vars.items():
            Validator.validate_string(value=var_name, name=f"Section '{section_name}' variable '{var_name}")
            Validator.validate_dict(value=var_obj, name=f"Section '{section_name}' variable '{var_name}'")

            var_type = var_obj.get("type")
            var_description = var_obj.get("description")
            var_value = var_obj.get("value")
            Validator.validate_string(value=var_type, name=f"Section '{section_name}' variable '{var_name}' type")
            Validator.validate_string_or_none(value=var_description, name=f"Section '{section_name}' variable '{var_name}' description")
            Validator.validate_string_or_none(value=var_value, name=f"Section '{section_name}' variable '{var_name}' default value")

            var = EnvVar(name=section.name + "_" + var_name, var_type=var_type, description=var_description, value=var_value)
            section.add_var(var)

            value = get_value_for_type(var_name=var.name, var_description=var.description, var_type=var.type, var_value=var.value)
            var.set_value(value)

    return root

def main() -> None:
    raw = load_schema_raw(SCHEMA_PATH)
    schema: EnvVarsRoot = parse_schema(raw)

    # Build a nicely formatted .env content from EnvVarsRoot
    lines: list[str] = []
    total_vars = 0

    for section in schema.sections:
        # Section header
        lines.append("#" * 120)
        lines.append(f"# {section.name}")
        if section.description:
            lines.append(f"# {section.description}")
        lines.append("#" * 120)

        # Variables within the section
        for var in section.vars:
            lines.append(var.get_dotenv_value())
            total_vars += 1

        # Blank line between sections
        lines.append("")

    content = "\n".join(lines).rstrip() + "\n"

    OUTPUT_ENV_PATH.write_text(content, encoding="utf-8")
    Printer.info(f"Wrote {OUTPUT_ENV_PATH} with {total_vars} entries.")


if __name__ == "__main__":
    main()