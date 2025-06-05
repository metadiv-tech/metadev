# MetaDev

A CLI tool to help developers with various development tasks.

## Installation

```bash
go install github.com/metadiv-tech/metadev@latest
```

## Commands

### `metadev i18n`

Extract translation keys from React TSX files and generate JSON translation files.

#### What it does

1. **Scans all `.tsx` files** in your project recursively
2. **Finds useTranslation declarations** like:
   ```tsx
   const { t: tCommon } = useTranslation('common')
   const { t: tAgent } = useTranslation("agent")
   ```
3. **Extracts translation keys** from function calls like:
   ```tsx
   tCommon('name')
   tCommon("description")
   tAgent("expire")
   tAgent('status')
   ```
4. **Groups keys by namespace** and generates JSON files with empty values
5. **Organizes output** in `.i18n/` directory
6. **Manages .gitignore** automatically

#### Example

Given this React component:
```tsx
import { useTranslation } from 'react-i18next';

export function UserProfile() {
  const { t: tCommon } = useTranslation('common');
  const { t: tUser } = useTranslation('user');

  return (
    <div>
      <h1>{tCommon('welcome')}</h1>
      <p>{tUser('profile_title')}</p>
      <button>{tCommon('save')}</button>
    </div>
  );
}
```

Running `metadev i18n` will generate:

**`.i18n/common.json`**
```json
{
  "save": "",
  "welcome": ""
}
```

**`.i18n/user.json`**
```json
{
  "profile_title": ""
}
```

#### Features

- **Smart directory skipping**: Automatically skips `node_modules`, `vendor`, `.git`, `.next`, `dist`, `build`
- **Automatic setup**: Creates `.i18n/` directory if it doesn't exist
- **Git integration**: Adds `.i18n/` to `.gitignore` automatically
- **Duplicate handling**: Prevents duplicate entries in translation files
- **Multiple namespace support**: Handles any number of translation namespaces

#### Usage

```bash
# Extract translation keys from current project
metadev i18n

# Output example:
# Created .gitignore and added .i18n/
# Generated /path/to/project/.i18n/common.json with 5 keys
# Generated /path/to/project/.i18n/user.json with 3 keys
# Successfully extracted 8 translation keys and generated translation files
```

#### AI Integration Workflow

After generating the JSON files with empty values, you can:

1. Pass the generated JSON files to AI tools
2. Ask AI to fill in appropriate translations
3. Replace the empty files with AI-generated translations

Example prompt for AI:
```
Please fill in the English translations for this JSON file:
{
  "welcome": "",
  "save": "",
  "profile_title": ""
}
```

## Project Structure

```
your-project/
├── src/
│   ├── components/
│   │   └── UserProfile.tsx
│   └── ...
├── .i18n/              # Generated translation files
│   ├── common.json
│   ├── user.json
│   └── ...
├── .gitignore          # Automatically updated
└── ...
```

## Development

```bash
# Clone the repository
git clone https://github.com/metadiv-tech/metadev.git
cd metadev

# Build the project
go build

# Run locally
./metadev i18n
```

## License

MIT License