import os
import shutil
from pathlib import Path
from typing import Iterable

from dotenv import load_dotenv


def _ensure_empty_directory(path: Path) -> None:
    """
    Ensure directory exists and is empty (remove all contents, keep directory).
    """
    path.mkdir(parents=True, exist_ok=True)
    for entry in path.iterdir():
        try:
            if entry.is_symlink() or entry.is_file():
                entry.unlink()
            elif entry.is_dir():
                shutil.rmtree(entry)
        except OSError as e:
            raise RuntimeError(f"Failed clearing '{entry}': {e}") from e


def _copy_dir_contents(src_dir: Path, dst_dir: Path) -> None:
    """
    Copy all items inside src_dir into dst_dir, preserving subdirectories.
    """
    if not src_dir.exists() or not src_dir.is_dir():
        raise ValueError(f"Source directory does not exist or is not a directory: {src_dir}")

    for item in src_dir.iterdir():
        dest_item = dst_dir / item.name
        if item.is_dir():
            shutil.copytree(item, dest_item, dirs_exist_ok=True)
        elif item.is_file() or item.is_symlink():
            # If symlink, copy the linked file content (like copy2 does)
            shutil.copy2(item, dest_item)
        else:
            # Skip FIFOs/devices, etc.
            pass

# TODO this is duplicated, refactor
def _get_required_env_var(var: str) -> str:
    """Get required environment variable"""
    try:
        return os.environ[var]
    except KeyError as e:
        missing = e.args[0] if e.args else str(e)
        raise ValueError(f"Missing required environment variable: {e}")

def _resolve_dest_from_env(env_key: str) -> Path:
    """
    Read an environment variable and resolve it to an absolute Path.
    Expands nested env vars and user (~).
    """
    raw = _get_required_env_var(env_key)

    expanded = os.path.expandvars(raw)
    expanded = os.path.expanduser(expanded)
    dest = Path(expanded).resolve()
    return dest


def _resolve_source_relative(src: str, base: Path) -> Path:
    """
    Resolve a repository-relative source path (e.g., './adguard/conf') from the script directory.
    """
    return (base / src).resolve()


def copy_conf_items(conf_list: Iterable[dict]) -> None:
    """
    Copy repository configuration files to environment-defined destinations.

    Each conf item must have:
      - 'source': a repository-relative directory path string
      - 'dest': the name of an environment variable that holds an absolute path
    """
    repo_root = Path(__file__).resolve().parents[1]

    # Load environment variables from the repo's .env file
    env_path = repo_root / ".env"
    if not env_path.exists():
        raise FileNotFoundError(f".env file not found at: {env_path}. Create it (you can start from .env.example).")
    load_dotenv(dotenv_path=env_path, override=True)

    for i, item in enumerate(conf_list, start=1):
        if not isinstance(item, dict):
            raise TypeError(f"conf_list item #{i} must be a dict, got: {type(item)}")

        if "source" not in item or "dest" not in item:
            raise ValueError(f"conf_list item #{i} must contain 'source' and 'dest' keys")

        src_rel = str(item["source"])
        dest_env = str(item["dest"])

        src_dir = _resolve_source_relative(src_rel, repo_root)
        dest_dir = _resolve_dest_from_env(dest_env)

        # Safety: prevent wiping the source directory by mistake
        if src_dir == dest_dir:
            raise ValueError(f"Source and destination resolve to the same directory: '{src_dir}'")

        print(f"[{i}] Copying from '{src_dir}' -> '{dest_dir}' (env: {dest_env})")

        # Prepare destination
        _ensure_empty_directory(dest_dir)

        # Copy contents
        _copy_dir_contents(src_dir, dest_dir)

        print(f"[{i}] Done.")


def main() -> None:
    # Safety warning and confirmation
    print("WARNING: This operation will REMOVE ALL CONTENTS from the destination configuration directories")
    print("and replace them with the configuration files from this repository.")
    confirm = input('Type "YES" to continue, or anything else to abort: ').strip()
    if confirm != "YES":
        print("Aborted by user.")
        return

    # Define your mappings here. Extend as needed. "source" must be a relative path to this repo's root.
    # "dest" must be the name of an environment variable that holds the absolute path of the config directory
    # of the corresponding service.
    conf_list = [
        {"source": "./adguard/conf", "dest": "ADGUARD_CONF_PATH"},
        # Examples you can enable/extend later:
        # {"source": "./traefik/config", "dest": "TRAEFIK_CONF_PATH"},
        # {"source": "./unbound/conf", "dest": "UNBOUND_CONF_PATH"},
    ]

    copy_conf_items(conf_list)


if __name__ == "__main__":
    main()