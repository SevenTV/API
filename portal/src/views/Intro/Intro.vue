<template>
	<main class="home">
		<div class="info">
			<div class="info-content">
				<div class="container"> 
					<div class="logo">
						<Logo />
					</div>
					<div class="info-text">
						<h1>Welcome to 7TV's Developer Portal!</h1>
						<p>
							 This area exists to provide documentation for developers to
							integrate with our API.
							<br />
							Head over to the docs page to get started!
						</p>
					</div>
				</div>
			</div>
		</div>
		<div class="page-body">
			<div class="home-content">
				<h1>In Development...</h1>		
			</div>
		</div>
		<div class="socials">
			<div class="social">
				<a class="social-split" href="https://discord.com/invite/2hhY2s3x" target="_blank">
					<Discord />
					<span>
						<p>Join 7TV on Discord</p>
						<i v-if="discord.count >= 0"><p>{{discord.count}} online now</p></i>
					</span>
				</a>
			</div>
			<div class="social">
				<a class="social-split" href="https://twitter.com/Official_7TV" target="_blank">
					<Twitter />
					<span>
						<p>Follow us on Twitter</p>
					</span>
				</a>
			</div>
			<div class="social">
				<a class="social-split" href="https://github.com/SevenTV" target="_blank">
					<Github />
					<span>
						<p>Contribute</p>
					</span>
				</a>
			</div>
		</div>
	</main>
</template>

<script setup lang="ts">
import { ref } from "vue";
import Logo from "@svg/Logo.vue";
import Discord from "@svg/Discord.vue";
import Twitter from "@svg/Twitter.vue";
import Github from "@svg/Github.vue";


const discord = ref({
	name: "",
	invite: "",
	count: -1,
});
{
	const req = new XMLHttpRequest();
	req.open("GET", "https://discord.com/api/guilds/817075418054000661/widget.json");
	req.onload = () => {
		const data = JSON.parse(req.responseText);
		discord.value.name = data.name;
		discord.value.invite = data.instant_invite;
		discord.value.count = data.presence_count;
	};
	req.send();
}

</script>

<style scoped lang="scss">
@import "@style/themes.scss";

main.home{
	position: relative;
	display: flex;
	flex-direction: column;
}
.info{
	display: grid;
	width: 100%;
	place-items: center;
	background-image: linear-gradient(25deg, #e12d2d, #f25ddc);;
	clip-path: polygon(100% 85%,0 100%,0 0,100% 0);
}

.container{
	display: flex;
	flex-grow: 1;
	justify-content: center;
	flex-wrap: wrap;
	align-items: center;
	padding-bottom: 5em;
}

.info-content{
	display: flex;
	flex-grow: 1;
	padding-top: 5%;
	justify-content: space-between;
	font-size: 1.25rem;
}

.info-text{
	display: flex;
	flex-direction: column;
	align-items: center;
	text-align: center;
	margin-left: 3em;
}

.logo{
	color: #c91d1d;
	font-size: 16em;
}

.info-text:nth-child(1){
	font-size: 3em;
	font-weight: 600;
	width: 8em;
	font-family: Work Sans,sans-serif;
}

.info-text:nth-child(2){
	font-size: 1.15em;
	font-weight: 400;
	width: 20em;
}

.page-body{
	display: flex;
	flex-grow: 1;
}

.home-content{
	display: flex;
	flex-direction: column;
	flex-grow: 1;
}

.home-content h1{
	color: #727272;
	margin-top: 2.5em;
	font-size: 2em;
	text-align: center;
}

.socials{
	display: grid;
	grid-template-columns: 1fr 1fr 1fr;
	background: #121717;
}

.social{
	padding: 1.5em 1em;
	display: grid;
	color: inherit;
	text-decoration: none;
}

.social-split{
	display: grid;
	grid-template-rows: 2em auto;
	gap: .5em;
	place-items: center;
	text-align: center;
	color: inherit;
}

@include breakpoint(lg, max) {

	.info{
		font-size: 1rem !important;
		flex-wrap: wrap;
		padding-bottom: 3.5em;

		.container{
			padding-top: 2.5em;
			padding-bottom: 2.5em !important;
		}
		.info-content{
			flex-direction: column;
		}

		.logo{
			font-size: 9em !important;
		}

		.info-text{
			margin-left: .5em;
		}

		.info-text:nth-child(1){
			font-size: 2em;
		}

		.info-text:nth-child(2){
			font-size: .9em;
			font-weight: 400;
			width: 18em;
		}
	}
	
}

</style>
