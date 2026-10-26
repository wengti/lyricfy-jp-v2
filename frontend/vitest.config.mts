import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
	plugins: [react()],
	resolve: {
		// Resolve "@/..." imports using the paths in tsconfig.json
		tsconfigPaths: true,
	},
	test: {
		// Simulated browser for component tests. Server-side tests opt into Node
		// with a `// @vitest-environment node` comment at the top of the file.
		environment: "jsdom",
		setupFiles: ["./vitest.setup.ts"],
		// Reset recorded mock calls, stubbed env vars and stubbed globals (e.g. fetch) between tests
		clearMocks: true,
		unstubEnvs: true,
		unstubGlobals: true,
	},
});
