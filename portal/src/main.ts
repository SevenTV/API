import { createApp, h } from "vue";
import { createHead } from "@vueuse/head";
import { createPinia } from "pinia";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import router from "@/router/router";
import App from "./App.vue";
import "./style.scss";

const app = createApp({
	render: () => h(App),
});

app.use(createHead()).use(createPinia()).use(router).component("font-awesome-icon", FontAwesomeIcon).mount("#app");
