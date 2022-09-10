import { defineConfig, loadEnv } from "vite";
import { resolve } from "path";
import vue from "@vitejs/plugin-vue";

// https://vitejs.dev/config/
export default ({ mode }) => {
	process.env = { ...process.env, ...loadEnv(mode, process.cwd()) };

	return defineConfig({
		build: {
			sourcemap: true,
			target: "es2021",
		},
		plugins: [vue()],
		resolve: {
			alias: {
				"@": resolve(__dirname, "src"),
				"@views": resolve(__dirname, "src/views"),
				"@style": resolve(__dirname, "src/assets/style"),
				"@svg": resolve(__dirname, "src/assets/svg"),
				"@components": resolve(__dirname, "src/components"),
			},
		},
	});
};
