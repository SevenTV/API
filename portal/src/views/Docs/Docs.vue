<template>
	<main class="docs">
		<div class="docs-sidebar">
			<!-- Version Selector -->
			<div class="sidebar-version-selector">
				<p class="sidebar-section">API HOST</p>
				<span>
					<span>{{ host.replace(/^(https?:|)\/\//, "") }}/</span>
					<select v-model="version">
						<option v-for="v of versions">{{ v }}</option>
					</select>
				</span>
			</div>

			<div class="sidebar-routes">
				<p class="sidebar-section">ROUTES</p>
				<div class="route-navigator">
					<div class="route-navigator-item" v-for="route in routes" :key="route.id">
						<a class="route-navigator-link" @click="scrollTo($event, route.id)">{{ route.summary }}</a>
					</div>
				</div>
			</div>
		</div>

		<div class="docs-content">
			<div v-if="error" class="load-error">
				<span>{{ error }}</span>
			</div>

			<div>
				<div class="route-list">
					<div class="route-list">
						<div class="route-item" v-for="route of routes">
							<h1 :id="route.id">
								<span>{{ route.summary }}</span>
								<span>
									<span class="route-item-method-name" :method="route.method">
										{{ route.method.toUpperCase() }}
									</span>
									{{ route.path }}
								</span>
							</h1>
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
				params: params as swag.Operation,
				method,
				path: key,
			});
		}
	}

	return result;
});

// computed(() => Object.keys(paths).map((k) => ({ name: k, ...paths[k] })));

const scrollTo = (evt: MouseEvent, id: string) => {
	const el = document.getElementById(id);
	if (!el) return;

	el.scrollIntoView({ behavior: "smooth" });
};

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
	params: swag.Operation;
}
</script>

<style scoped lang="scss">
@import "@style/themes.scss";

main.docs {
	display: grid;
	grid-template-columns: max(14em, min(18em, 25%)) 1fr;
	scroll-behavior: smooth;

	@include themify() {
		.docs-sidebar {
			background-color: lighten(themed("backgroundColor"), 2);
		}

		.route-item > h1 {
			background-color: darken(themed("backgroundColor"), 1.5);
		}

		.route-item-method-name {
			&[method="get"] {
				color: themed("accent");
			}
			&[method="post"] {
				color: themed("primary");
			}
			&[method="patch"] {
				color: adjust-hue(themed("primary"), 95);
			}
			&[method="put"] {
				color: adjust-hue(themed("warning"), 25);
			}
			&[method="delete"] {
				color: themed("warning");
			}
		}
	}

	.docs-sidebar {
		display: flex;
		flex-direction: column;
		background-color: red;
		padding-left: 1em;
		padding-top: 1em;
		gap: 1em;
		position: relative;
		max-height: 100vh;
		overflow: auto;

		.sidebar-section {
			font-size: 0.85rem;
			font-weight: 600;
		}

		> .sidebar-version-selector > select {
			margin-top: 0.5em;
		}
	}
	.docs-content {
		.route-list {
			display: grid;
			gap: 2.5em;
		}
		.route-item {
			> h1 {
				display: grid;
				align-items: center;
				padding: 0.25em;

				> span:nth-child(2) {
					font-size: 0.85rem;
					font-weight: 400;
					margin-left: 0.1em;

					> .route-item-method-name {
						font-weight: 600;
					}
				}
			}
		}
	}
}
</style>
