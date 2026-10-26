import type { NextConfig } from "next";

const nextConfig: NextConfig = {
	// Self-contained server build for the Docker image (Cloud Run)
	output: "standalone", //  traces which files the app actually uses at runtime and copies only those into .next/standalone/
	cacheComponents: true,
	partialPrefetching: true,
	turbopack: {
		rules: {
			"*.css": {
				loaders: ["@tailwindcss/turbopack"],
				as: "*.css",
			},
		},
	},
};

export default nextConfig;
