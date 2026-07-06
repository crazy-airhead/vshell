const availableFlags = new Set([
  'ae', 'af', 'ai', 'ao', 'ar', 'at', 'au', 'az', 'bb', 'bd', 'be', 'bg', 'bh', 'bm',
  'bn', 'bo', 'br', 'bs', 'by', 'ca', 'cf', 'cg', 'ch', 'ck', 'cl', 'cm', 'cn', 'co',
  'cr', 'cu', 'cz', 'de', 'dj', 'dk', 'do', 'dz', 'ec', 'ee', 'eg', 'es', 'fi', 'fj',
  'fr', 'gb', 'ge', 'gf', 'gh', 'gm', 'gn', 'gr', 'gu', 'gy', 'hk', 'hn', 'ht', 'hu',
  'id', 'ie', 'il', 'in', 'iq', 'ir', 'is', 'it', 'jm', 'jp', 'ke', 'kg', 'kh', 'kp',
  'kr', 'kw', 'kz', 'la', 'lb', 'lr', 'lt', 'lu', 'ly', 'ma', 'mc', 'md', 'mg', 'ml',
  'mn', 'mo', 'mt', 'mu', 'mv', 'mx', 'my', 'mz', 'na', 'ng', 'nl', 'no', 'np', 'nr',
  'nz', 'om', 'pa', 'pe', 'ph', 'pk', 'pl', 'pt', 'py', 'qa', 'ro', 'ru', 'sa', 'sb',
  'sc', 'sd', 'se', 'sg', 'sk', 'sl', 'sn', 'so', 'sr', 'sv', 'sy', 'tg', 'th', 'tj',
  'tm', 'tn', 'tr', 'tw', 'tz', 'ua', 'ug', 'un', 'us', 'uy', 'uz', 've', 'vn', 'ye',
  'za', 'zm', 'zw',
])

type IpRange = {
  start: number
  end: number
  country: string
}

const ranges: IpRange[] = [
  range('1.0.0.0', '1.1.255.255', 'au'),
  range('1.12.0.0', '1.15.255.255', 'cn'),
  range('1.32.0.0', '1.32.255.255', 'hk'),
  range('1.34.0.0', '1.34.255.255', 'tw'),
  range('1.36.0.0', '1.39.255.255', 'hk'),
  range('1.64.0.0', '1.79.255.255', 'hk'),
  range('1.112.0.0', '1.115.255.255', 'jp'),
  range('5.8.0.0', '5.8.255.255', 'ru'),
  range('8.0.0.0', '8.255.255.255', 'us'),
  range('13.32.0.0', '13.59.255.255', 'us'),
  range('13.112.0.0', '13.115.255.255', 'jp'),
  range('13.124.0.0', '13.125.255.255', 'kr'),
  range('13.228.0.0', '13.229.255.255', 'sg'),
  range('13.248.0.0', '13.255.255.255', 'us'),
  range('14.0.0.0', '14.31.255.255', 'jp'),
  range('14.102.0.0', '14.102.255.255', 'hk'),
  range('18.128.0.0', '18.255.255.255', 'us'),
  range('20.0.0.0', '20.255.255.255', 'us'),
  range('23.0.0.0', '23.255.255.255', 'us'),
  range('27.0.0.0', '27.31.255.255', 'cn'),
  range('27.96.0.0', '27.111.255.255', 'hk'),
  range('31.0.0.0', '31.255.255.255', 'gb'),
  range('34.64.0.0', '34.127.255.255', 'us'),
  range('35.0.0.0', '35.255.255.255', 'us'),
  range('36.0.0.0', '36.255.255.255', 'cn'),
  range('39.96.0.0', '39.108.255.255', 'cn'),
  range('40.0.0.0', '40.255.255.255', 'us'),
  range('43.128.0.0', '43.159.255.255', 'sg'),
  range('45.0.0.0', '45.255.255.255', 'us'),
  range('47.52.0.0', '47.91.255.255', 'hk'),
  range('47.92.0.0', '47.127.255.255', 'cn'),
  range('47.128.0.0', '47.129.255.255', 'sg'),
  range('47.235.0.0', '47.246.255.255', 'hk'),
  range('49.0.0.0', '49.255.255.255', 'cn'),
  range('52.0.0.0', '52.95.255.255', 'us'),
  range('52.192.0.0', '52.199.255.255', 'jp'),
  range('52.220.0.0', '52.221.255.255', 'sg'),
  range('54.0.0.0', '54.255.255.255', 'us'),
  range('58.0.0.0', '58.255.255.255', 'cn'),
  range('60.0.0.0', '60.255.255.255', 'cn'),
  range('62.0.0.0', '62.255.255.255', 'gb'),
  range('66.0.0.0', '66.255.255.255', 'us'),
  range('74.0.0.0', '74.255.255.255', 'us'),
  range('78.0.0.0', '78.255.255.255', 'gb'),
  range('80.0.0.0', '95.255.255.255', 'gb'),
  range('101.32.0.0', '101.33.255.255', 'sg'),
  range('101.64.0.0', '101.95.255.255', 'cn'),
  range('103.0.0.0', '103.255.255.255', 'hk'),
  range('104.0.0.0', '104.255.255.255', 'us'),
  range('106.0.0.0', '106.255.255.255', 'cn'),
  range('108.0.0.0', '108.255.255.255', 'us'),
  range('110.0.0.0', '113.255.255.255', 'cn'),
  range('118.0.0.0', '119.255.255.255', 'cn'),
  range('120.0.0.0', '123.255.255.255', 'cn'),
  range('128.0.0.0', '128.255.255.255', 'us'),
  range('129.146.0.0', '129.159.255.255', 'us'),
  range('134.0.0.0', '134.255.255.255', 'us'),
  range('139.0.0.0', '139.255.255.255', 'cn'),
  range('140.82.0.0', '140.82.255.255', 'us'),
  range('142.250.0.0', '142.251.255.255', 'us'),
  range('143.92.0.0', '143.92.255.255', 'hk'),
  range('150.109.0.0', '150.109.255.255', 'sg'),
  range('152.32.0.0', '152.32.255.255', 'hk'),
  range('154.0.0.0', '154.255.255.255', 'za'),
  range('157.0.0.0', '157.255.255.255', 'jp'),
  range('159.0.0.0', '159.255.255.255', 'us'),
  range('161.0.0.0', '161.255.255.255', 'us'),
  range('162.0.0.0', '162.255.255.255', 'us'),
  range('172.64.0.0', '172.71.255.255', 'us'),
  range('175.0.0.0', '175.255.255.255', 'cn'),
  range('180.0.0.0', '183.255.255.255', 'cn'),
  range('185.0.0.0', '185.255.255.255', 'gb'),
  range('188.0.0.0', '188.255.255.255', 'ru'),
  range('192.0.0.0', '192.255.255.255', 'us'),
  range('198.0.0.0', '198.255.255.255', 'us'),
  range('202.0.0.0', '203.255.255.255', 'cn'),
  range('208.0.0.0', '208.255.255.255', 'us'),
  range('216.0.0.0', '216.255.255.255', 'us'),
]

export function flagForHost(host: string) {
  const country = countryForIPv4(host)
  return flagForCountry(country)
}

export function flagForCountry(country: string | null | undefined) {
  const flag = country && availableFlags.has(country) ? country : 'un'
  return `/flags/${flag}.png`
}

export function countryForIPv4(host: string) {
  const value = parseIPv4(host)
  if (value === null || isPrivateIPv4(value)) return null
  const found = ranges.find(item => value >= item.start && value <= item.end)
  return found?.country ?? null
}

function range(start: string, end: string, country: string): IpRange {
  return { start: parseIPv4(start) ?? 0, end: parseIPv4(end) ?? 0, country }
}

function parseIPv4(host: string) {
  const normalized = host.trim()
  if (!/^\d{1,3}(\.\d{1,3}){3}$/.test(normalized)) return null
  const parts = normalized.split('.').map(Number)
  if (parts.some(part => part < 0 || part > 255)) return null
  return ((parts[0] * 256 + parts[1]) * 256 + parts[2]) * 256 + parts[3]
}

function isPrivateIPv4(value: number) {
  return (
    inRange(value, '10.0.0.0', '10.255.255.255') ||
    inRange(value, '172.16.0.0', '172.31.255.255') ||
    inRange(value, '192.168.0.0', '192.168.255.255') ||
    inRange(value, '127.0.0.0', '127.255.255.255') ||
    inRange(value, '169.254.0.0', '169.254.255.255')
  )
}

function inRange(value: number, start: string, end: string) {
  const startValue = parseIPv4(start) ?? 0
  const endValue = parseIPv4(end) ?? 0
  return value >= startValue && value <= endValue
}
