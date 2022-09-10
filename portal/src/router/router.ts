import { createRouter, createWebHistory, RouteRecordRaw } from "vue-router";

const routes = [
	{
		path: "/",
		redirect: "/docs",
		// component: () => import("@views/Intro/Intro.vue"),
	},
	{
		path: "/docs",
		component: () => import("@views/Docs/Docs.vue"),
	},
] as RouteRecordRaw[];

const router = createRouter({
	history: createWebHistory(import.meta.env.BASE_URL),
	routes,
});

export default router;
