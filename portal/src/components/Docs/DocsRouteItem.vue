<template>
	<div class="routeItemParent">
		<h1 :id="route.id">
			<span style="margin-bottom: 0.188rem">{{ route.summary }}</span>
		</h1>
		<div style="padding: 0 6px 6px 6px; font-size: large">
			<span class="route-item-method-name" :method="route.method">
				{{ route.method.toUpperCase() }}
			</span>
			<span v-if="isSupported">
				<div class="route-path" title="Copy URL" @click="copy(`${host}/${version}${route.path}`)">
					<pre style="display: inline">{{ route.path }}</pre>
				</div>
			</span>
			<span v-else>
				<span class="route-path">
					{{ route.path }}
				</span>
			</span>
			<br />
			<div style="margin-top: 0.625rem">
				<i>{{ route.description }}</i>
				<br />
				<div class="route-parameter-box">
					<div v-for="f in route.params.parameters" v-if="route.params.parameters != null">
						<h2 v-if="f?.name">{{ f?.name }}</h2>
						string
						<!-- change this soon, don't want this to be hardcoded in, but the object doesn't have a type apparently? -->
						<br />
						<small>
							<i v-if="f?.in">({{ f?.in }})</i>
						</small>
						<br />
						<p style="font-weight: bold; margin-top: 0.2rem" v-if="f?.description">{{ f?.description }}</p>
						<pre v-if="route.params.consumes">
                            {{ route.params.consumes }}
                        </pre>
					</div>
					<div v-else>
						Accepts:
						<pre v-if="route.params.consumes">
                            {{ route.params.consumes }}
                        </pre>
					</div>
				</div>
				<small v-if="route.method !== 'get'">
                    Parameter type: {{ route.params.produces?.join(", ") ?? "not specified" }}
                </small>
				<br v-if="route.method !== 'get'" />
				<small>tags: {{ route.params.tags?.join(", ") ?? "none" }}</small>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">

// Copy to clipboard stuff - lets users copy the request url to their clipboard with ease
import { useClipboard } from "@vueuse/core";

const { isSupported, copy } = useClipboard();

defineProps<{
	route: any;
	host: string;
	version: string;
}>();

</script>

<style scoped lang="scss">

@import "@style/themes.scss";

pre {
	margin-top: 0.4rem;
	overflow-x: auto;
	white-space: pre-wrap;
	word-wrap: break-word;
	white-space: normal;
}

.routeItemParent {
	padding: 0.5rem;
	@include themify() {
		.route-parameter-box {
			background-color: darken(themed("backgroundColor"), 1);
			padding: 0.625rem;
			margin-top: 0.625rem;
			margin-bottom: 0.625rem;
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

		.route-item-method-name {
			font-size: medium;
			color: white;
			padding-left: 0.313rem;
			padding-right: 0.313rem;
			border-radius: 0.188rem;

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
				background-color: darken(adjust-hue(themed("warning"), 25), 10);
			}

			&[method="delete"] {
				background-color: #d7392e;
			}
		}
	}
}

</style>
