import { defineConfig } from "vite";
import { resolve } from "path";
import vue from "@vitejs/plugin-vue";

// https://vitejs.dev/config/
export default defineConfig({
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
