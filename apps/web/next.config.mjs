/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Habilita el output standalone para el Dockerfile de producción.
  // Genera .next/standalone con solo los archivos necesarios (~50MB vs ~500MB).
  output: 'standalone',
};

export default nextConfig;
