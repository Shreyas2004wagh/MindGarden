/** @type {import('next').NextConfig} */
const nextConfig = {
  turbopack: {
    root: __dirname,
  },
  images: { unoptimized: true },
};

module.exports = nextConfig;
