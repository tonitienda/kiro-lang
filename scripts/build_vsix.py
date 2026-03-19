#!/usr/bin/env python3
import json
import shutil
import sys
import tempfile
import zipfile
from pathlib import Path
from xml.sax.saxutils import escape


def usage() -> None:
    print('usage: scripts/build_vsix.py <extension-dir> <output-file>', file=sys.stderr)


def collect_files(ext_dir: Path):
    include_roots = [
        ext_dir / 'package.json',
        ext_dir / 'README.md',
        ext_dir / 'extension.js',
        ext_dir / 'language-configuration.json',
        ext_dir / 'syntax',
        ext_dir / 'node_modules',
    ]
    for root in include_roots:
        if not root.exists():
            raise FileNotFoundError(f'missing required VS Code extension path: {root}')
    for root in include_roots:
        if root.is_file():
            yield root, Path('extension') / root.name
            continue
        for path in sorted(root.rglob('*')):
            if path.is_dir() or any(part.startswith('.') for part in path.relative_to(root).parts):
                continue
            yield path, Path('extension') / path.relative_to(ext_dir)


def build_manifest(package: dict) -> str:
    display_name = escape(package.get('displayName', package['name']))
    description = escape(package.get('description', ''))
    tags = escape(','.join(package.get('keywords', [])))
    categories = escape(','.join(package.get('categories', [])))
    engine = escape(package['engines']['vscode'])
    publisher = escape(package['publisher'])
    extension_id = escape(f"{package['publisher']}.{package['name']}")
    version = escape(package['version'])
    return f'''<?xml version="1.0" encoding="utf-8"?>
<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011">
  <Metadata>
    <Identity Language="en-US" Id="{extension_id}" Version="{version}" Publisher="{publisher}" />
    <DisplayName>{display_name}</DisplayName>
    <Description xml:space="preserve">{description}</Description>
    <Tags>{tags}</Tags>
    <Categories>{categories}</Categories>
    <GalleryFlags>Public</GalleryFlags>
    <Properties>
      <Property Id="Microsoft.VisualStudio.Code.Engine" Value="{engine}" />
    </Properties>
  </Metadata>
  <Installation>
    <InstallationTarget Id="Microsoft.VisualStudio.Code" Version="{engine}" />
  </Installation>
  <Dependencies />
  <Assets>
    <Asset Type="Microsoft.VisualStudio.Code.Manifest" Path="extension/package.json" />
    <Asset Type="Microsoft.VisualStudio.Services.Content.Details" Path="extension/README.md" />
  </Assets>
</PackageManifest>
'''


def build_content_types() -> str:
    return '''<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="json" ContentType="application/json" />
  <Default Extension="js" ContentType="application/javascript" />
  <Default Extension="md" ContentType="text/markdown" />
  <Default Extension="vsixmanifest" ContentType="text/xml" />
  <Default Extension="xml" ContentType="text/xml" />
</Types>
'''


def main() -> int:
    if len(sys.argv) != 3:
        usage()
        return 1

    ext_dir = Path(sys.argv[1]).resolve()
    out_file = Path(sys.argv[2]).resolve()
    package = json.loads((ext_dir / 'package.json').read_text())

    out_file.parent.mkdir(parents=True, exist_ok=True)
    temp_dir = Path(tempfile.mkdtemp(prefix='kiro-vsix-'))
    try:
        staging = temp_dir / 'staging'
        staging.mkdir(parents=True, exist_ok=True)
        (staging / '[Content_Types].xml').write_text(build_content_types())
        (staging / 'extension.vsixmanifest').write_text(build_manifest(package))

        for src, dest in collect_files(ext_dir):
            target = staging / dest
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, target)

        with zipfile.ZipFile(out_file, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
            for path in sorted(staging.rglob('*')):
                if path.is_dir():
                    continue
                archive.write(path, path.relative_to(staging).as_posix())
    finally:
        shutil.rmtree(temp_dir, ignore_errors=True)

    print(f'created {out_file}')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
