<template>
	<main class="docs">
		<DocsSideBarVue :host="host.replace(/^(https?:|)\/\//, '')" :versions="versions" :routes="routes.result" />
		<div class="docs-content">
			<div v-if="error" class="load-error">
				<span>{{ error }}</span>
			</div>
			<div>
				<div class="route-list">
					<div class="route-list">
						<div class="route-item" :method="route.method" v-for="route of routes.result">
							<DocsRouteItemVue :route="route" :host="host" :version="version" />
						</div>
					</div>
				</div>
			</div>
		</div>
	</main>
</template>

<script setup lang="ts">
import swag from "swagger-schema-official";
import { computed, ref } from "vue";
import DocsSideBarVue from "@/components/Docs/DocsSideBar.vue";
import DocsRouteItemVue from "@/components/Docs/DocsRouteItem.vue";

const error = ref("");

const host = import.meta.env.VITE_APP_API_REST ?? "";
const versions = ref<string[]>(["v3"]);
const version = ref("v3");

const req = await fetch(`${host}/${version.value}/docs`);

const spec = (await req.json()) as swag.Spec;

const info = spec.info;
const basePath = spec.basePath;
const schemes = spec.schemes;
const paths = spec.paths;
const definitions = spec.definitions;

// Get every path split by method
const routes = computed(() => {
	const result = [] as RouteDef[];
	const keys = Object.keys(paths);

	for (const key of keys) {
		const path = paths[key];
		const methods = Object.keys(path);

		for (const method of methods) {
			const params = path[method as "get" | "post" | "put" | "delete"];
			result.push({
				id: method + " " + key,
				summary: params?.summary ?? "",
				description: params?.description ?? "",
				params: params as swag.Spec,
				method,
				path: key,
			});
		}
	}

	return {
		result,
	};
});

// computed(() => Object.keys(paths).map((k) => ({ name: k, ...paths[k] })));

const methodSort = {
	get: 1,
	post: 2,
	patch: 3,
	put: 4,
	delete: 5,
};

interface RouteDef {
	id: string;
	summary: string;
	description: string;
	method: string;
	path: string;
	params: swag.Spec; // changed to swag.Spec because it fit our needs better than swag.Operation
}
</script>

<style scoped lang="scss">
@import "@style/themes.scss";

#app {
	width: 100%;
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-direction: column;
	padding: 1rem;
}

main.docs {
	@include breakpoint(md, max) {
		display: flex;
	}
	@include breakpoint(md, min) {
		display: grid;
	}
	grid-template-columns: max(14em, min(18em, 25%)) 1fr;
	scroll-behavior: smooth;

	@include themify() {
		.route-item {
			background-color: darken(themed("backgroundColor"), 5);
			margin-left: 1rem;
			margin-right: 1rem;
			padding: 5px;
			transition: all 0.5s ease-in-out;

			@include breakpoint(md, max) {
				width: calc(100vw - 30px);
			}

			@include breakpoint(lg, min) {
				width: calc(100vw - 320px);
			}

			&[method="get"] {
				box-shadow: lighten(themed("accent"), 1) -3px 0;
			}

			&[method="post"] {
				box-shadow: lighten(themed("primary"), 1) -3px 0;
			}

			&[method="patch"] {
				box-shadow: adjust-hue(themed("primary"), 95) -3px 0;
			}

			&[method="put"] {
				box-shadow: darken(adjust-hue(themed("warning"), 25), 10) -3px 0;
			}

			&[method="delete"] {
				box-shadow: #d7392e -3px 0;
			}
		}

		.route-item > h1 {
			border-radius: 0.313rem;
		}
	}

	.docs-content {
		.route-list {
			margin-top: 0.625rem;
			margin-bottom: 0.625rem;
			display: grid;
			gap: 1.5em;
		}

		.route-item {
			> h1 {
				display: grid;
				align-items: center;
				padding: 0.25em;

				> span:nth-child(2) {
					font-size: 1rem;
					font-weight: 400;
					margin-left: 0.1em;
				}
			}
		}
	}
}
</style>
