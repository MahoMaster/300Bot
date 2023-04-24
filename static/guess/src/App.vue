<template>
	<div class="hby">
		<GuessGift
			ref="guessgiftDom"
			:turn-number="20"
			:turns-frequency="300"
			@end-turns="endTurns"
		>
		</GuessGift>
		<button
			v-if="!hasStart"
			type="primary"
			@click="gameStart"
		>
			开始
		</button>
	</div>
</template>

<script setup>
import { GuessGift } from "./dist/nutbig.es.js";
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
const code = ref("");
const progress = ref(0);
if (!params.hasOwnProperty("code")) {
	alert("参数错误");
	location.replace("http://www.mahomaster.com");
	// return;
} else {
	code.value = params.code;
	axios(`/lumaCode2Info?code=${code.value}&mode=3`).then(res => {
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
			alert("你需要找出金豆在哪个碗里");
		}
	});
}
const guessgiftDom = ref(null);

const hasStart = ref(false);
const clickFlag = ref(false);
const gameStart = () => {
	hasStart.value = true;
	guessgiftDom.value.start();
	setTimeout(() => {
		report(0);
		alert("这么久不点?突破失败！");
	}, 80000);
};

const endTurns = flag => {
	clickFlag.value = true;
	console.log("抽奖结束");
	console.log(flag);
	if (flag) {
		progress.value = 100;
		report(progress.value);
		alert("突破成功");
	} else {
		progress.value = 0;
		report(progress.value);
		alert("突破失败");
	}
};

const report = p => {
	axios(`/lumaReport?code=${code.value}&progress=${p}&mode=3`).then(res => {
		console.log(res);
	});
};
</script>
<style>
.hby {
	width: 100%;
	height: 100%;
	display: flex;
	flex-direction: column;
	justify-content: center;
	align-items: center;
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
