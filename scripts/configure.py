#!/usr/bin/env python3
"""
Generate a .env.<TIMESTAMP> file by reading the entries from the JSON configuration file
"""

import json
import time
from pathlib import Path
from typing import NotRequired, TypedDict

from jsonschema import validate

from scripts.env_vars import (
    EnvVar,
    EnvVarsRoot,
    EnvVarsSection,
    TypeHandlerRegistry,
    get_value_for_type,
)
from scripts.printer import Printer

# Resolve paths relative to the repository root (parent of scripts/)
_REPO_ROOT = Path(__file__).resolve().parents[1]
OUTPUT_ENV_PATH = _REPO_ROOT / f".env.generated.{int(time.time())}"

class ConfigVariable(TypedDict):
    name: str
    type: str
    description: str
    value: NotRequired[str]

class ConfigSection(TypedDict):
    name: str
    description: str
    variables: list[ConfigVariable]

class ConfigRoot(TypedDict):
    prefix: str
    sections: list[ConfigSection]

def validate_and_load_config() -> ConfigRoot:
    conf_file_path = _REPO_ROOT / "scripts" / "env.config.json"
    conf_schema_file_path = _REPO_ROOT / "scripts" / "env.config.schema.json"
    if not conf_file_path.exists() or not conf_file_path.is_file():
        raise ValueError(f"Config file not found at {conf_file_path.resolve()}")
    if not conf_schema_file_path.exists() or not conf_schema_file_path.is_file():
        raise ValueError(f"Schema of config file not found at {conf_schema_file_path.resolve()}")

    with conf_file_path.open("r", encoding="utf-8") as conf_file, conf_schema_file_path.open("r", encoding="utf-8") as conf_schema_file:
            json_config = json.load(conf_file)
            json_config_schema = json.load(conf_schema_file)
            validate(instance=json_config, schema=json_config_schema)
            return json_config

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


def parse_config(config_root: ConfigRoot,*, registry: TypeHandlerRegistry | None = None) -> EnvVarsRoot:
    root = EnvVarsRoot(prefix=config_root["prefix"])

    for config_section in config_root["sections"]:
        section_full_name = f"{root.prefix}_{config_section["name"]}"
        section = EnvVarsSection(name=section_full_name, description=config_section["description"])
        root.add_section(section)

        Printer.info(f"\n\n>>>>>>>>>> Section: {section.name}")
        if section.description:
            for line in Printer.wrap_lines(text=section.description, width=120):
                Printer.info(line)

        for config_var in config_section["variables"]:
            var = EnvVar(
                name=f"{section.name}_{config_var["name"]}",
                type=config_var["type"],
                description=config_var["description"],
                value=config_var.get("value", None),
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
    config_root = validate_and_load_config()
    schema: EnvVarsRoot = parse_config(config_root)

    builder = DotenvBuilder()
    for section in schema.sections:
        builder.add_section(section)

    content = builder.build()
    OUTPUT_ENV_PATH.write_text(content, encoding="utf-8")
    Printer.info(f"Wrote {OUTPUT_ENV_PATH} with {builder.total_vars} entries.")


if __name__ == "__main__":
    main()
