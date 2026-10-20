const base = process.env.VITE_API_BASE_URL
if (process.env.TF_ANDROID_LOCAL_PREVIEW === '1') {
  if (base) {
    console.error('La vista previa local usa el mismo origen de Docker; deja VITE_API_BASE_URL vacío.')
    process.exit(1)
  }
  process.exit(0)
}
if (!base) {
  console.error('Configura VITE_API_BASE_URL con el origen HTTPS de la API antes de sincronizar Android.')
  process.exit(1)
}
let url
try {
  url = new URL(base)
} catch {
  console.error('VITE_API_BASE_URL debe ser una URL HTTPS válida.')
  process.exit(1)
}
if (url.protocol !== 'https:' || url.pathname !== '/' || url.search || url.hash) {
  console.error('VITE_API_BASE_URL debe ser solo el origen HTTPS, sin ruta ni parámetros.')
  process.exit(1)
}
