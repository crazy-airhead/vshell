const BINARY_EXTENSIONS = new Set([
  // archives
  '.zip', '.tar', '.gz', '.bz2', '.xz', '.7z', '.rar', '.tgz', '.zst',
  // executables
  '.exe', '.app', '.dmg', '.msi', '.deb', '.rpm', '.apk',
  // media
  '.mp3', '.mp4', '.avi', '.mkv', '.mov', '.wmv', '.flac', '.ogg', '.wav', '.webm', '.aac',
  // images
  '.png', '.jpg', '.jpeg', '.gif', '.bmp', '.ico', '.webp', '.svg', '.tiff', '.tif',
  // databases
  '.db', '.sqlite', '.sqlite3',
  // ISOs
  '.iso',
  // fonts
  '.ttf', '.otf', '.woff', '.woff2',
  // compiled
  '.o', '.so', '.dll', '.dylib', '.class', '.pyc', '.bin', '.wasm',
  // PDF
  '.pdf',
])

const MAX_EDIT_SIZE = 5 * 1024 * 1024 // 5MB

export function isEditableFile(filename: string, fileSize: number): boolean {
  if (fileSize > MAX_EDIT_SIZE) return false
  const dotIdx = filename.lastIndexOf('.')
  const ext = dotIdx >= 0 ? filename.slice(dotIdx).toLowerCase() : ''
  if (BINARY_EXTENSIONS.has(ext)) return false
  return true
}

const EXT_TO_LANG: Record<string, string> = {
  '.ts': 'typescript', '.tsx': 'typescript',
  '.js': 'javascript', '.jsx': 'javascript',
  '.json': 'json',
  '.html': 'html', '.htm': 'html',
  '.css': 'css', '.scss': 'scss', '.less': 'less',
  '.vue': 'html',
  '.md': 'markdown',
  '.yaml': 'yaml', '.yml': 'yaml',
  '.xml': 'xml',
  '.go': 'go',
  '.py': 'python',
  '.java': 'java',
  '.c': 'c', '.h': 'c',
  '.cpp': 'cpp', '.cc': 'cpp', '.cxx': 'cpp', '.hpp': 'cpp',
  '.rs': 'rust',
  '.rb': 'ruby',
  '.php': 'php',
  '.sh': 'shell', '.bash': 'shell', '.zsh': 'shell',
  '.sql': 'sql',
  '.toml': 'ini',
  '.ini': 'ini', '.cfg': 'ini', '.conf': 'ini',
  '.dockerfile': 'dockerfile',
  '.tf': 'hcl',
  '.lua': 'lua',
  '.swift': 'swift',
  '.kt': 'kotlin',
  '.scala': 'scala',
  '.r': 'r',
  '.dart': 'dart',
}

export function detectLanguage(filename: string): string {
  const dotIdx = filename.lastIndexOf('.')
  const ext = dotIdx >= 0 ? filename.slice(dotIdx).toLowerCase() : ''
  if (EXT_TO_LANG[ext]) return EXT_TO_LANG[ext]
  const base = filename.toLowerCase()
  if (base === 'makefile' || base === 'gnumakefile') return 'makefile'
  if (base === 'dockerfile') return 'dockerfile'
  if (base === 'cmakelists.txt') return 'cmake'
  return 'plaintext'
}
