<template>
	<div class="hby">
		<nutbig-giftrain
			ref="rain"
			class="rain"
			width="100%"
			height="100%"
			rain-time="60000"
			rain-num="4"
			@gameOver="gameOver"
			@start="start"
			@click="click"
		>
		</nutbig-giftrain>
		<!-- <div
			v-if="!isStart"
			class="start"
			@click="onStart"
		>
			开始
		</div>-->
		<div class="num">{{ num }}</div>
	</div>
</template>

<script setup>
import axios from "axios";
const getUrlVars = () => {
	var vars = [],
		hash;
	var decodeUrl = decodeURI(window.location.href);
	var hashes = window.location.href.slice(window.location.href.indexOf("?") + 1).split("&");
	for (var i = 0; i < hashes.length; i++) {
		hash = hashes[i].split("=");
		vars.push(hash[0]);
		vars[hash[0]] = hash[1];
	}
	return vars;
};
// console.log(getUrlVars());
const params = getUrlVars();
const need = ref(0);
const num = ref(0);
const code = ref("");
const progress = ref(0);
if (!params.hasOwnProperty("code")) {
	alert("参数错误");
	location.replace("http://www.mahomaster.com");
	// return;
} else {
	code.value = params.code;
	axios(`/lumaCode2Info?code=${code.value}&mode=1`).then(res => {
		if (res.data.code != 0) {
			alert(res.data.msg);
			location.replace("http://www.mahomaster.com");
		} else {
			while (true) {
				let qq = prompt("请输入qq号");
				if (qq != res.data.data.qq) {
					alert("qq不正确");
				} else {
					break;
				}
			}
			need.value = res.data.data.need_rank;
			alert("你需要在60s内点击到" + need.value + "个红包来完成突破");
			onStart();
		}
	});
}
const rain = ref();
const isStart = ref(false);
const flag = {
	f10: false,
	f30: false,
	f60: false,
	f90: false,
	f100: false
};
const gameOver = () => {
	// console.log("游戏结束");
	isStart.value = false;
	if (num.value < need.value) {
		report(0);
		alert("你没有完成突破");
		num.value = 0;
	}
};
const start = () => {
	isStart.value = true;
};
const report = p => {
	axios(`/lumaReport?code=${code.value}&progress=${p}&mode=2`).then(res => {
		console.log(res);
	});
};
const checkProgress = () => {
	if (progress.value == 10 && !flag.f10) {
		flag.f10 = true;
		report(progress.value);
	}
	if (progress.value == 30 && !flag.f30) {
		flag.f30 = true;
		report(progress.value);
	}
	if (progress.value == 60 && !flag.f60) {
		flag.f60 = true;
		report(progress.value);
	}
	if (progress.value == 90 && !flag.f90) {
		flag.f90 = true;
		report(progress.value);
	}
	if (progress.value == 100 && !flag.f100) {
		flag.f100 = true;
		report(progress.value);
	}
	// if (
	// 	progress.value == 10 ||
	// 	progress.value == 30 ||
	// 	progress.value == 60 ||
	// 	progress.value == 90 ||
	// 	progress.value == 100
	// ) {
	// 	report(progress.value);
	// }
};
const click = () => {
	// console.log("点击");
	num.value++;
	progress.value = Math.floor((num.value / need.value) * 100);
	console.log(progress.value);
	if (num.value >= need.value) {
		progress.value = 100;
		alert("恭喜你完成突破");
		gameOver();
	}
	checkProgress();
};
const onStart = () => {
	rain.value.startRain();
};
</script>
<style>
.hby {
	width: 100%;
	height: 100%;
}
.rain {
	margin-top: 0 !important;
	width: 100%;
	height: 100%;
}
.nutbig-giftrain .nutbig-giftrain-content {
	background: url("//img13.360buyimg.com/imagetools/jfs/t1/156139/35/24533/600373/61974f3eEf612507c/88df16bece0b202f.png")
		no-repeat;
	background-size: 100% 100%;
	position: relative;
}
.start {
	width: 100px;
	height: 30px;
	background: linear-gradient(
		135deg,
		rgba(114, 60, 255, 1) 0%,
		rgba(111, 58, 255, 1) 63.49938195167575%,
		rgba(150, 110, 255, 1) 87.35307751528254%,
		rgba(149, 117, 241, 1) 100%
	);
	border-radius: 10px;
	display: flex;
	justify-content: center;
	align-items: center;
	color: rgba(255, 255, 255, 1);
	position: absolute;
	bottom: 0;
	left: 40%;
}
.num {
	position: absolute;
	right: 10px;
	color: #fff;
	font-weight: 30px;
	bottom: 50%;
}
</style>
