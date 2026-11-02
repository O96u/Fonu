import { resolveCountryGeo } from './worldMapCoords'

/** Province center [lng, lat] for fly lines — keys match ip2region-normalized names. */
export const PROVINCE_COORDS: Record<string, [number, number]> = {
  北京: [116.405285, 39.904989],
  天津: [117.190182, 39.125596],
  上海: [121.472644, 31.231706],
  重庆: [107.304962, 29.533155],
  河北: [114.502461, 38.045474],
  山西: [111.849248, 36.857014],
  辽宁: [123.429096, 41.796767],
  吉林: [125.3245, 43.886841],
  黑龙江: [128.642464, 46.756967],
  江苏: [119.767413, 33.041544],
  浙江: [120.153576, 29.287459],
  安徽: [117.283042, 31.26119],
  福建: [118.306239, 26.075302],
  江西: [115.592151, 27.676493],
  山东: [118.000923, 36.275807],
  河南: [113.665412, 33.757975],
  湖北: [114.298572, 30.684355],
  湖南: [111.782279, 28.09409],
  广东: [113.280637, 23.125178],
  广西: [108.320004, 22.82402],
  海南: [109.83119, 19.031971],
  四川: [104.065735, 30.659462],
  贵州: [106.713478, 26.578343],
  云南: [101.512251, 24.740609],
  西藏: [89.132212, 30.860361],
  陕西: [108.948024, 34.263161],
  甘肃: [103.823557, 36.058039],
  青海: [96.778916, 35.623178],
  宁夏: [106.278179, 37.26637],
  新疆: [85.617733, 40.792818],
  内蒙古: [111.670801, 41.818311],
  香港: [114.173355, 22.320048],
  澳门: [113.54909, 22.198951],
  台湾: [121.509062, 24.044332],
}

/** 地级市中心。ip2region 常把「成都市」放在省份列、城市列写成运营商。 */
const CITY_COORDS: Record<string, [number, number]> = {
  成都: [104.066541, 30.572269],
  绵阳: [104.679127, 31.467459],
  德阳: [104.397894, 31.126855],
  宜宾: [104.643215, 28.751768],
  南充: [106.110698, 30.837793],
  泸州: [105.442258, 28.871811],
  达州: [107.468023, 31.209571],
  乐山: [103.765568, 29.552106],
  自贡: [104.778442, 29.33903],
  内江: [105.058433, 29.580228],
  遂宁: [105.592898, 30.532847],
  广元: [105.843357, 32.435435],
  广安: [106.633369, 30.455962],
  眉山: [103.848538, 30.075439],
  资阳: [104.627636, 30.128901],
  雅安: [103.013261, 29.980537],
  巴中: [106.747477, 31.867903],
  攀枝花: [101.718637, 26.582347],
  广州: [113.264385, 23.129112],
  深圳: [114.085947, 22.547],
  珠海: [113.553986, 22.224979],
  佛山: [113.122717, 23.028762],
  东莞: [113.746262, 23.046237],
  中山: [113.382391, 22.521113],
  惠州: [114.412599, 23.079404],
  汕头: [116.708463, 23.37102],
  江门: [113.094942, 22.590431],
  湛江: [110.364977, 21.274898],
  武汉: [114.305393, 30.593099],
  西安: [108.940175, 34.341568],
  郑州: [113.625368, 34.7466],
  济南: [117.000923, 36.675807],
  青岛: [120.355173, 36.082982],
  南京: [118.796877, 32.060255],
  苏州: [120.619585, 31.299379],
  杭州: [120.15507, 30.274084],
  宁波: [121.550357, 29.874556],
  合肥: [117.283042, 31.86119],
  福州: [119.306239, 26.075302],
  厦门: [118.11022, 24.490474],
  南昌: [115.892151, 28.676493],
  长沙: [112.982279, 28.19409],
  海口: [110.33119, 20.031971],
  三亚: [109.508268, 18.247872],
  南宁: [108.320004, 22.82402],
  昆明: [102.712251, 25.040609],
  贵阳: [106.713478, 26.578343],
  拉萨: [91.132212, 29.660361],
  兰州: [103.823557, 36.058039],
  西宁: [101.778916, 36.623178],
  银川: [106.278179, 38.46637],
  乌鲁木齐: [87.617733, 43.792818],
  呼和浩特: [111.670801, 40.818311],
  哈尔滨: [126.642464, 45.756967],
  长春: [125.3245, 43.886841],
  沈阳: [123.429096, 41.796767],
  大连: [121.618622, 38.91459],
  石家庄: [114.502461, 38.045474],
  太原: [112.549248, 37.857014],
  张家口: [114.884091, 40.811901],
  常州: [119.946973, 31.772752],
  无锡: [120.301663, 31.574729],
  济宁: [116.587245, 35.415393],
  重庆: [106.504962, 29.533155],
  北京: [116.405285, 39.904989],
  上海: [121.472644, 31.231706],
  天津: [117.190182, 39.125596],
  香港: [114.173355, 22.320048],
  澳门: [113.54909, 22.198951],
  台北: [121.509062, 25.044332],
}

export type ServerGeo = {
  country?: string
  province?: string
  city?: string
  label?: string
}

function lookupPlace(name: string): [number, number] | null {
  const base = stripAdminSuffix(name)
  if (!base || base === '未知' || base === '海外') return null
  if (CITY_COORDS[base]) return CITY_COORDS[base]
  return resolveProvinceCoord(base)
}

/** 本机落点：优先城市，其次省份。对不上就不画，避免退回全国几何中心。 */
export function resolveServerHub(node: ServerGeo | null | undefined): [number, number] | null {
  if (!node) return null
  const country = (node.country ?? '').trim()
  const province = (node.province ?? '').trim()
  const city = (node.city ?? '').trim()
  if (country && country !== '中国' && country !== '本地') {
    return resolveCountryGeo(country)?.coord ?? null
  }
  const fromLabel = (node.label ?? '')
    .split('·')
    .map((part) => part.trim())
    .filter(Boolean)
  for (const name of [city, province, ...fromLabel]) {
    const at = lookupPlace(name)
    if (at) return at
  }
  return null
}

/** 城市 / 简称 → 省级键（与 GeoJSON `properties.name` 一致） */
const REGION_TO_PROVINCE: Record<string, keyof typeof PROVINCE_COORDS> = {
  广州: '广东',
  深圳: '广东',
  石家庄: '河北',
  张家口: '河北',
  郑州: '河南',
  北京: '北京',
  上海: '上海',
}

function stripAdminSuffix(raw: string): string {
  const p = raw.trim()
  const suffixes = ['特别行政区', '维吾尔自治区', '壮族自治区', '回族自治区', '自治区', '省', '市']
  for (const s of suffixes) {
    if (p.endsWith(s)) return p.slice(0, -s.length) || p
  }
  return p
}

export function resolveProvinceCoord(name: string): [number, number] | null {
  const key = name.trim()
  if (!key || key === '未知' || key === '海外') return null
  const base = stripAdminSuffix(key)
  const mapped = REGION_TO_PROVINCE[key] ?? REGION_TO_PROVINCE[base]
  if (mapped && PROVINCE_COORDS[mapped]) return PROVINCE_COORDS[mapped]
  if (PROVINCE_COORDS[key]) return PROVINCE_COORDS[key]
  if (PROVINCE_COORDS[base]) return PROVINCE_COORDS[base]
  for (const [k, v] of Object.entries(PROVINCE_COORDS)) {
    if (key.includes(k) || k.includes(key) || base.includes(k) || k.includes(base)) return v
  }
  return null
}
