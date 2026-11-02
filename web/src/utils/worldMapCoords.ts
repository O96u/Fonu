/** Country → ECharts world map feature name + centroid for fly lines. */
export type CountryGeo = { mapName: string; coord: [number, number] }

export const COUNTRY_GEO: Record<string, CountryGeo> = {
  中国: { mapName: 'China', coord: [104.195397, 35.86166] },
  美国: { mapName: 'United States', coord: [-98.5795, 39.8283] },
  日本: { mapName: 'Japan', coord: [138.252924, 36.204824] },
  德国: { mapName: 'Germany', coord: [10.451526, 51.165691] },
  英国: { mapName: 'United Kingdom', coord: [-3.435973, 55.378051] },
  法国: { mapName: 'France', coord: [2.213749, 46.227638] },
  印度: { mapName: 'India', coord: [78.96288, 20.593684] },
  俄罗斯: { mapName: 'Russia', coord: [105.318756, 61.52401] },
  巴西: { mapName: 'Brazil', coord: [-51.92528, -14.235004] },
  加拿大: { mapName: 'Canada', coord: [-106.346771, 56.130366] },
  澳大利亚: { mapName: 'Australia', coord: [133.775136, -25.274398] },
  韩国: { mapName: 'Korea', coord: [127.766922, 35.907757] },
  新加坡: { mapName: 'Singapore', coord: [103.819836, 1.352083] },
  荷兰: { mapName: 'Netherlands', coord: [5.291266, 52.132633] },
  意大利: { mapName: 'Italy', coord: [12.56738, 41.87194] },
  西班牙: { mapName: 'Spain', coord: [-3.74922, 40.463667] },
  波兰: { mapName: 'Poland', coord: [19.145136, 51.919438] },
  瑞典: { mapName: 'Sweden', coord: [18.643501, 60.128161] },
  瑞士: { mapName: 'Switzerland', coord: [8.227512, 46.818188] },
  土耳其: { mapName: 'Turkey', coord: [35.243322, 38.963745] },
  墨西哥: { mapName: 'Mexico', coord: [-102.552784, 23.634501] },
  印度尼西亚: { mapName: 'Indonesia', coord: [113.921327, -0.789275] },
  泰国: { mapName: 'Thailand', coord: [100.992541, 15.870032] },
  越南: { mapName: 'Vietnam', coord: [108.277199, 14.058324] },
  马来西亚: { mapName: 'Malaysia', coord: [101.975766, 4.210484] },
  菲律宾: { mapName: 'Philippines', coord: [121.774017, 12.879721] },
  乌克兰: { mapName: 'Ukraine', coord: [31.16558, 48.379433] },
  伊朗: { mapName: 'Iran', coord: [53.688046, 32.427908] },
  沙特阿拉伯: { mapName: 'Saudi Arabia', coord: [45.079162, 23.885942] },
  阿联酋: { mapName: 'United Arab Emirates', coord: [53.847818, 23.424076] },
  南非: { mapName: 'South Africa', coord: [22.937506, -30.559482] },
  埃及: { mapName: 'Egypt', coord: [30.802498, 26.820553] },
  尼日利亚: { mapName: 'Nigeria', coord: [8.675277, 9.081999] },
  阿根廷: { mapName: 'Argentina', coord: [-63.616672, -38.416097] },
  智利: { mapName: 'Chile', coord: [-71.542969, -35.675147] },
  哥伦比亚: { mapName: 'Colombia', coord: [-74.297333, 4.570868] },
  新西兰: { mapName: 'New Zealand', coord: [174.885971, -40.900557] },
  芬兰: { mapName: 'Finland', coord: [25.748151, 61.92411] },
  挪威: { mapName: 'Norway', coord: [8.468946, 60.472024] },
  丹麦: { mapName: 'Denmark', coord: [9.501785, 56.26392] },
  比利时: { mapName: 'Belgium', coord: [4.469936, 50.503887] },
  奥地利: { mapName: 'Austria', coord: [14.550072, 47.516231] },
  捷克: { mapName: 'Czech Rep.', coord: [15.472962, 49.817492] },
  爱尔兰: { mapName: 'Ireland', coord: [-8.24389, 53.41291] },
  葡萄牙: { mapName: 'Portugal', coord: [-8.224454, 39.399872] },
  希腊: { mapName: 'Greece', coord: [21.824312, 39.074208] },
  以色列: { mapName: 'Israel', coord: [34.851612, 31.046051] },
  巴基斯坦: { mapName: 'Pakistan', coord: [69.345116, 30.375321] },
  孟加拉国: { mapName: 'Bangladesh', coord: [90.356331, 23.684994] },
  中国香港: { mapName: 'China', coord: [114.173355, 22.320048] },
  中国台湾: { mapName: 'China', coord: [121.509062, 25.044332] },
  中国澳门: { mapName: 'China', coord: [113.54909, 22.198951] },
}

const ENGLISH_COUNTRY_TO_CN: Record<string, string> = {}
for (const [cn, geo] of Object.entries(COUNTRY_GEO)) {
  ENGLISH_COUNTRY_TO_CN[geo.mapName] = cn
}
ENGLISH_COUNTRY_TO_CN['United States of America'] = '美国'
ENGLISH_COUNTRY_TO_CN['USA'] = '美国'
ENGLISH_COUNTRY_TO_CN['UK'] = '英国'

/** Map English geo_country values to 中文 keys used in COUNTRY_GEO. */
export function normalizeCountryKey(name: string): string {
  const n = name.trim()
  if (!n) return n
  if (COUNTRY_GEO[n]) return n
  if (ENGLISH_COUNTRY_TO_CN[n]) return ENGLISH_COUNTRY_TO_CN[n]
  const lower = n.toLowerCase()
  for (const [en, cn] of Object.entries(ENGLISH_COUNTRY_TO_CN)) {
    if (en.toLowerCase() === lower) return cn
  }
  return n
}

export function resolveCountryGeo(country: string): CountryGeo | null {
  const key = normalizeCountryKey(country)
  if (!key || key === '未知') return null
  return COUNTRY_GEO[key] ?? null
}

export function mapFeatureDisplayName(mapName: string): string {
  return ENGLISH_COUNTRY_TO_CN[mapName] ?? mapName
}
