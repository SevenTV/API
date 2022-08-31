import { createApp, h } from "vue";
import { createHead } from "@vueuse/head";
import { createPinia } from "pinia";
import router from "@/router/router";
import App from "./App.vue";
import "./style.scss";

const app = createApp({
	render: () => h(App),
});

app.use(createHead()).use(createPinia()).use(router).mount("#app");
