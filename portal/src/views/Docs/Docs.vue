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
				<button class="toggle-collapse" @click="toggleNav">Expand Links</button>

			</div>
			<div v-if="routeLinkNavOpen" class="sidebar-routes-big">
				<p class="sidebar-section">ROUTES</p>
				<div class="route-navigator">
					<div class="route-navigator-item" v-for="route in routes.result" :key="route.id">
						<a class="route-navigator-link" @click="scrollTo($event, route.id)">{{ route.summary }}</a>
					</div>
				</div>
			</div>
			<div id="routeNavigationToggle" class="sidebar-routes-small">
				<p class="sidebar-section">ROUTES</p>
				<div class="route-navigator">
					<div class="route-navigator-item" v-for="route in routes.result" :key="route.id">
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
						<div class="route-item" :method="route.method" v-for="route of routes.result">
							<h1 :id="route.id">
								<span style="margin-bottom:3px;">{{ route.summary }}</span>
							</h1>
							<div style="padding: 0 6px 6px 6px; font-size: large;">
								<span class="route-item-method-name" :method="route.method">
									{{ route.method.toUpperCase() }}
								</span>
								<span v-if="isSupported">
									<div class="route-path" title="Copy URL"
										@click="copy(`${host}/${version}${route.path}`)">
										<pre style="display: inline;">{{ route.path }}</pre>
									</div>
								</span>
								<span v-else>
									<span class="route-path">
										{{ route.path }}
									</span>
								</span>
								<br />
								<div style="margin-top: 10px;">
									<i>{{ route.description }}</i>
									<br />
									<div class="route-parameter-box">
										<div v-for="f in route.params.parameters"
											v-if="route.params.parameters != null">
											<h2>{{ f?.name ?? 'no name' }}</h2>
											string <!-- change this soon, don't want this to be hardcoded in, but the object doesn't have a type apparently? -->
											<br />
											<small><i>({{ f?.in ?? 'no data input type' }})</i></small>
											<br />
											<p style="font-weight: bold;">{{ f?.description ?? 'no description' }}</p>
											<pre
												style="margin-top: 5px;">{{ route.params.consumes ?? 'no filetype data' }}</pre>
										</div>
										<div v-else>
											Accepts:
											{{ route.params.consumes ?? "no data" }}
										</div>
									</div>
									<small v-if="route.method !== 'get'">Parameter type:
										{{ route.params.produces?.join(", ") ?? 'not specified' }}</small>
									<br v-if="route.method !== 'get'" />
									<small>tags: {{ route.params.tags?.join(", ") ?? 'none' }}</small>
								</div>
							</div>
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
import { useClipboard, usePermission } from '@vueuse/core'

const routeLinkNavOpen = ref(false);

const toggleNav = () => {
	routeLinkNavOpen.value = !routeLinkNavOpen.value;
};

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
		result // i'm too lazy to take this out of the object...
	}
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
	params: swag.Spec; // changed to swag.Spec because it fit our needs better than swag.Operation
}

// Copy to clipboard stuff - lets users copy the request url to their clipboard with ease
const { text, copied, isSupported, copy } = useClipboard()

</script>

<style scoped lang="scss">

// TODO clean up these style defs in the future

@import "@style/themes.scss";

body {
	margin: 0;
	max-width: 100vw;
}

dl {
	margin: 0 0 1em;
	;

	dt {
		background-color: #ccc;
		padding: 1em;
		font-weight: bold;
	}

	dd {
		padding: 0;
		margin: 0;
		border: 1px solid #ccc;
		border-top: 0;
		padding: 1em;
	}
}



#app {
	width: 100vw;
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-direction: column;
	padding: 16px;
}


.accordions {
	width: 100%;
	max-width: 500px;
}

pre {
	overflow-x: auto;
	/* Use horizontal scroller if needed */
	white-space: pre-wrap;
	/* css-3 */
	white-space: -moz-pre-wrap !important;
	/* Mozilla, since 1999 */
	word-wrap: break-word;
	/* Internet Explorer 5.5+ */
	white-space: normal;
}

main.docs {
	display: grid;
	grid-template-columns: max(14em, min(18em, 25%)) 1fr;
	scroll-behavior: smooth;

	@include breakpoint(md, max) {
		display: block;

		.docs-sidebar {
			padding-bottom: 1rem;
		}

		.toggle-collapse {
			display: inline;
		}
	}

	@include themify() {
		.docs-sidebar {
			position: sticky;
			top: 4.5rem;
			background-color: lighten(themed("backgroundColor"), 2);
		}

		.route-parameter-box {
			background-color: darken(themed("backgroundColor"), 1);
			padding: 10px;
			margin-top: 10px;
			margin-bottom: 10px;
		}

		.route-path {
			margin-left: 5px;
			border-radius: 4px;
			padding: 2px 10px 2px 10px;
			cursor: pointer;
			transition: all 0.05s ease-in-out;
			display: inline;
		}

		.route-path:hover {
			text-decoration: underline;
		}

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

		.route-item>h1 {
			border-radius: 5px;
		}

		.route-item-method-name {
			font-size: medium;
			color: white;
			padding-left: 5px;
			padding-right: 5px;
			border-radius: 3px;

			&[method="get"] {
				background-color: lighten(themed("accent"), 1);
			}

			&[method="post"] {
				background-color: lighten(themed("primary"), 1);
			}

			&[method="patch"] {
				background-color: adjust-hue(themed("primary"), 95);
			}

			&[method="put"] {
				background-color: darken(adjust-hue(themed("warning"), 25), 10)
			}

			&[method="delete"] {
				background-color: #d7392e
			}
		}

		.route-item-description {
			border-radius: 0 0 5px 5px;
			padding: 10px;
			color: white;
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
		max-height: calc(100vh - 4.5rem);
		overflow: auto;

		.sidebar-routes-big {

			// this is just to keep it hidden on default for smaller screens. JS will hide/unhide it.
			@include breakpoint(md, min) {
				display: none;
			}
		}

		.sidebar-routes-small {

			// this is just to keep it hidden on default for smaller screens. JS will hide/unhide it.
			@include breakpoint(md, max) {
				display: none;
			}
		}

		.sidebar-section {
			font-size: 0.85rem;
			font-weight: 600;
		}

		>.sidebar-version-selector>select {
			margin-top: 0.5em;
		}
	}

	@include themify() {
		.route-navigator-link {
			cursor: pointer;
			display: block;
			padding: 10px;
			margin: 5px 15px 5px 0px;
			color: themed("color");
			background-color: themed("backgroundColor");

			.route-navigator-link:hover {
				background-color: themed("secondaryBackgroundColor");
			}
		}
	}



	.route-navigator-item {
		transition: all 0.1s ease-in-out;
		border: red;

		:hover {
			border-left: 4px solid red;
			border-bottom: 4px solid red;
			transition: all 0.1s ease-in-out;
		}
	}

	.docs-content {
		.route-list {
			margin-top: 10px;
			margin-bottom: 10px;
			display: grid;
			gap: 1.5em;
		}

		.route-item {
			>h1 {
				display: grid;
				align-items: center;
				padding: 0.25em;

				>span:nth-child(2) {
					font-size: 1rem;
					font-weight: 400;
					margin-left: 0.1em;

					>.route-item-method-name {
						font-weight: 600;
					}
				}
			}
		}
	}
}

.toggle-collapse {
	display: none;
	background-color: #373737;
	font: inherit;
	color: inherit;
	margin-left: 5px;
	border: 0.1em solid transparent;
	padding: 0.3em;
	border-radius: 0.5em;
	place-self: center;

	&:hover {
		border-color: #373737;
	}

	&:active {
		background-color: #515151;
	}
}
</style>
