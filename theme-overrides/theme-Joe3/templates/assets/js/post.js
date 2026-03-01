/**鏂囩珷椤甸€昏緫 */
const postContext = {
	limited: false,
	/* 鍒濆鍖栬瘎璁哄悗鍙 */
	// initReadLimit() {
	// 	if (
	// 		PageAttrs.metas_enable_read_limit &&
	// 		PageAttrs.metas_enable_read_limit.trim() !== "true"
	// 	)
	// 		return;
	// 	postContext.limited = true;
	// 	const $article = $(".page-post .joe_detail__article");
	// 	const $content = $("#post-inner");
	// 	const $hideMark = $(".page-post .joe_read_limited");
	// 	const clientHeight =
	// 		document.documentElement.clientHeight || document.body.clientHeight;
	// 	const cid = $(".joe_detail").attr("data-cid");
	//
	// 	// 绉婚櫎闄愬埗
	// 	const removeLimit = () => {
	// 		postContext.limited = false;
	// 		$hideMark.parent().remove();
	// 		$article.removeClass("limited");
	// 		postContext.initToc(true); // 閲嶆柊娓叉煋TOC
	// 	};
	//
	// 	// 濡傛灉鏂囩珷鍐呭楂樺害灏忎簬绛変簬灞忓箷楂樺害锛屽垯鑷姩蹇界暐闄愬埗
	// 	if ($content.height() < clientHeight + 180) {
	// 		removeLimit();
	// 		return;
	// 	}
	//
	// 	// 妫€鏌ユ湰鍦扮殑 partialIds
	// 	const checkPartialIds = (postId, cb) => {
	// 		const localIds = localStorage.getItem("partialIds");
	// 		if (localIds && localIds.includes(postId)) {
	// 			// console.log("宸茬粡璇勮杩囦簡");
	// 			removeLimit(); // 绉婚櫎闄愬埗
	// 		} else {
	// 			cb && cb();
	// 		}
	// 	};
	//
	// 	// 鏇存柊褰撳墠璇勮鐘舵€?	// 	const updateState = async () => {
	// 		// console.log("璇勮鎴愬姛锛屾洿鏂扮姸鎬?);
	// 		const scrollTop = $hideMark.offset().top - 180;
	// 		const localIds = localStorage.getItem("partialIds");
	//
	// 		await Utils.sleep(800); // 寤惰繜涓€涓?	// 		removeLimit(); // 绉婚櫎闄愬埗
	// 		localStorage.setItem("partialIds", localIds ? localIds + "," + cid : cid); // 璁板綍id
	// 		Qmsg.success("鎰熻阿鎮ㄧ殑鏀寔");
	//
	// 		// 婊氬姩鍒板師浣嶇疆
	// 		$("html,body").animate(
	// 			{
	// 				scrollTop,
	// 			},
	// 			500
	// 		);
	// 	};
	//
	// 	// 鐩戝惉璇勮鎴愬姛浜嬩欢锛堝尯鍒嗛娆″拰鍚庣画鎻愪氦锛?	// 	const handleCallback = () => {
	// 		// console.log("娌℃湁璇勮璁板綍");
	// 		const commentNode = document.getElementsByTagName("halo-comment")[0];
	// 		commentNode.addEventListener("post-success", (_data) => {
	// 			// console.log(_data, "璇勮鎴愬姛");
	// 			// 妫€鏌ユ槸鍚﹀凡缁忚瘎璁鸿繃璇ユ枃绔?	// 			checkPartialIds(cid, updateState);
	// 		});
	// 	};
	//
	// 	checkPartialIds(cid, handleCallback);
	// },
	/* 鏂囩珷澶嶅埗 + 鐗堟潈鏂囧瓧 */
	initCopy() {
		if (PageAttrs.metas_enable_copy === "false" || !ThemeConfig.enable_copy)
			return;

		const curl = location.href;
		const author = $(".joe_detail").attr("data-author");
		const postTitle = $(".joe_detail .joe_detail__title").text();
		const postDescription = $('html head meta[name=description]').attr('content');

		$(".joe_detail__article").on("copy", function (e) {
			const selection = window.getSelection();
			const selectionText = selection.toString().replace(/<宸茶嚜鍔ㄦ姌鍙?/g, "");

			const appendLink = (ThemeConfig.enable_copy_right_text
				? ThemeConfig.copy_right_text ||
				`\r\n\r\n====================================\r\n鏂囩珷浣滆€咃細 ${author}\r\n鏂囩珷鏉ユ簮锛?${ThemeConfig.blog_title}(${ThemeConfig.blog_url})\r\n鏂囩珷鏍囬锛?${postTitle}\r\n鏂囩珷閾炬帴锛?${curl}\r\n鐗堟潈澹版槑锛?鍐呭閬靛惊 CC 4.0 BY-SA 鐗堟潈鍗忚锛岃浆杞借闄勪笂鍘熸枃鍑哄閾炬帴鍙婃湰澹版槑銆俙
				: "")
				.replace(/{postUrl}/g, curl)
				.replace(/{postTitle}/g, postTitle)
				.replace(/{postAuthor}/g, author)
				.replace(/{BlogTitle}/g, ThemeConfig.blog_title)
				.replace(/{BlogUrl}/g, ThemeConfig.blog_url)
				.replace(/{postDescription}/g, postDescription);
				
			if (window.clipboardData) {
				const copytext = selectionText + appendLink;
				window.clipboardData.setData("Text", copytext);
				return false;
			} else {
				const body_element = document.body;
				const copytext = selectionText + appendLink;
				const newdiv = document.createElement("pre");
				newdiv.style.position = "absolute";
				newdiv.style.left = "-99999px";
				body_element.appendChild(newdiv);
				newdiv.innerText = copytext;
				selection.selectAllChildren(newdiv);
				setTimeout(function () {
					body_element.removeChild(newdiv);
				}, 0);
			}
		});
	},
	/* 鍒濆鍖栨枃绔犲垎浜?*/
	initShare() {
		if (PageAttrs.metas_enable_share === "false" || !ThemeConfig.enable_share)
			return;
		if (ThemeConfig.enable_share_link && $(".icon-share-link").length) {
			$(".icon-share-link").each((_index, item) => {
				const shareLinkEle = $(item)[0];
				const postTitle = shareLinkEle.dataset.postTitle;
				const postDescription = shareLinkEle.dataset.postDescription;
				const postAuthor = shareLinkEle.dataset.postAuthor;
				const template = shareLinkEle.dataset.template;
				
				// 鑷畾涔夊垎浜摼鎺ユā鐗堬紝鏀寔鍙橀噺 {postUrl}銆亄postTitle}銆亄postAuthor}銆亄postDescription}锛岀暀绌哄垯浣跨敤榛樿妯＄増(鏂囩珷閾炬帴)锛屼緥濡? 鏂囩珷鍒嗕韩: {postAuthor} 鍙戝竷浜嗘枃绔犮€恵postTitle}銆戯紝閾炬帴: {postUrl}
				let copyContent = window.location.href;
				if (template) {
					copyContent = template.replace(/{postUrl}/g, location.href)
						.replace(/{postTitle}/g, postTitle)
						.replace(/{postAuthor}/g, postAuthor)
						.replace(/{postDescription}/g, postDescription)
						.replace(/{BlogTitle}/g, ThemeConfig.blog_title)
						.replace(/{BlogUrl}/g, ThemeConfig.blog_url);

					if (!/{postUrl}/.test(template)) { // 濡傛灉妯＄増涓病鏈墈postUrl}鍙橀噺锛屽垯闇€瑕佽拷鍔犳枃绔犻摼鎺?						copyContent += ` 锛屾枃绔犻摼鎺? ${location.href}`;
					}
				}

				new ClipboardJS(shareLinkEle, {
					text: () => copyContent,
				}).on("success", () => Qmsg.success("鏂囩珷閾炬帴宸插鍒?));
			});
		}
		if (ThemeConfig.enable_share_weixin && $(".qrcode_wx").length) {
			$(".qrcode_wx").qrcode({
				width: 140,
				height: 140,
				render: "canvas",
				typeNumber: -1,
				correctLevel: 0,
				background: "#ffffff",
				foreground: "#000000",
				text: location.href,
			});
		}
	},
	/* 鏂囩珷鐐硅禐 */
	initLike() {
		if (
			PageAttrs.metas_enable_like === "false" ||
			!ThemeConfig.enable_like ||
			!$(".joe_detail__agree").length
		)
			return;
		const $detail = $(".joe_detail");
		const cid = $detail.attr("data-cid");
		const viewer = ($detail.attr("data-viewer") || "anonymousUser").trim();
		const agreeStoreKey = encryption(`agree:${viewer}`);
		let likes = +($detail.attr("data-clikes") || 0);
		const loadAgreeList = () => {
			try {
				const raw = localStorage.getItem(agreeStoreKey);
				if (!raw) return [];
				const decoded = decrypt(raw);
				const parsed = JSON.parse(decoded);
				return Array.isArray(parsed) ? parsed : [];
			} catch (_err) {
				return [];
			}
		};
		const saveAgreeList = (arr) => {
			localStorage.setItem(agreeStoreKey, encryption(JSON.stringify(arr)));
		};
		let agreeArr = loadAgreeList();
		let flag = agreeArr.includes(cid);
		const $icons = $(".joe_detail__agree, .post-operate-like");
		const $iconLike = $icons.find(".icon-like");
		const $iconUnlike = $icons.find(".icon-unlike");
		const $likeNum = $icons.find(".nums");
		const renderLikeState = () => {
			$iconLike.removeClass("active");
			$iconUnlike.removeClass("active");
			$icons.removeClass("active");
			if (flag) {
				$iconUnlike.addClass("active");
				$icons.addClass("active");
			} else {
				$iconLike.addClass("active");
			}
			$likeNum.html(likes).show();
		};
		renderLikeState();

		let _loading = false;
		$iconLike.add($iconUnlike).on("click", function (e) {
			e.stopPropagation();
			if (_loading) return;
			_loading = true;
			agreeArr = loadAgreeList();
			flag = agreeArr.includes(cid);

			$.ajax({
				url: "/apis/api.halo.run/v1alpha1/trackers/upvote",
				type: "post",
				contentType: "application/json; charset=utf-8",
				data: JSON.stringify({
					group: "content.halo.run",
					plural: "posts",
					name: cid,
				}),
			})
				.then((_res) => {
					if (flag) {
						likes = Math.max(0, likes - 1);
						const index = agreeArr.findIndex((_) => _ === cid);
						if (index >= 0) agreeArr.splice(index, 1);
						flag = false;
					} else {
						likes += 1;
						if (!agreeArr.includes(cid)) agreeArr.push(cid);
						flag = true;
					}
					saveAgreeList(agreeArr);
					renderLikeState();
				})
				.catch((_err) => {
					// keep current UI state when request failed
				})
				.finally(() => {
					_loading = false;
				});
		});
	},
	/* 鏂囩珷鐩綍 */
	initToc(reload) {
		if (
			PageAttrs.metas_enable_toc === "false" ||
			!ThemeConfig.enable_toc ||
			!$(".toc-container").length
		)
			return;

		// 鍘熷鍐呭鐨勬枃绔犱笉鏀寔TOC
		if (PageAttrs.metas_use_raw_content === "true") {
			$("#js-toc").html(
				"<div class=\"toc-nodata\">鏆備笉鏀寔瑙ｆ瀽鍘熷鍐呭鐩綍</div>"
			);
			$(".toc-container").show();
			return;
		}

		// 鍥炲鍙鐨勬枃绔犻娆′笉娓叉煋TOC
		if (
			PageAttrs.metas_enable_read_limit === "true" &&
			!reload &&
			postContext.limited
		) {
			$("#js-toc").html(
				"<div class=\"toc-nodata\">鏂囩珷鍐呭涓嶅畬鏁达紝鐩綍浠呰瘎璁哄悗鍙</div>"
			);
			$(".toc-container").show();
			return;
		}

		// 娓叉煋TOC&澶勭悊鐩稿叧浜や簰
		const $html = $("html");
		const $mask = $(".joe_header__mask");
		const $btn_mobile_toc = $(".joe_action .toc");
		const $mobile_toc = $(".joe_header__toc");
		const $tocContainer = $("#js-toc");
		const $tocMobileContainer = $("#js-toc-mobile");

		// 鍒濆鍖朤OC
		tocbot.init({
			tocSelector: Joe.isMobile ? "#js-toc-mobile" : "#js-toc",
			contentSelector: ".joe_detail__article",
			ignoreSelector: ".js-toc-ignore",
			headingSelector: "h1,h2,h3,h4,h5,h6",
			collapseDepth: +(PageAttrs.metas_toc_depth || ThemeConfig.toc_depth || 0),
			scrollSmooth: true,
			includeTitleTags: true,
			// scrollSmoothDuration: 400,
			hasInnerContainers: false,
			headingsOffset: 80, // 鐩綍涓珮浜殑鍋忕Щ鍊硷紝鍜宻crollSmoothOffset鏈夊叧鑱?			scrollSmoothOffset: -80, // 灞忓箷婊氬姩鐨勫亸绉诲€硷紙杩欓噷鍜屽鑸潯鍥哄畾涔熸湁鍏宠仈锛?			positionFixedSelector: ".toc-container", // 鍥哄畾绫绘坊鍔犵殑瀹瑰櫒
			positionFixedClass: "is-position-fixed", // 鍥哄畾绫诲悕绉?			fixedSidebarOffset: "auto",
			// disableTocScrollSync: false,
			onClick: function (e) {
				// console.log(e);
				if (Joe.isMobile) {
					// 鏇存柊绉诲姩绔痶oc鏂囩珷婊氬姩浣嶇疆
					$html.removeClass("disable-scroll");
					$mobile_toc.removeClass("active");
					$mask.removeClass("active slideout");
					// if (location.hash) {
					// 	$("html,body").animate(
					// 		{
					// 			scrollTop: $(decodeURIComponent(location.hash)).offset().top,
					// 		},
					// 		0
					// 	);
					// }
				}

				window.tocPhase = true;
			},
			scrollEndCallback: function (e) {
				// console.log(e);
				window.tocPhase = null;
			},
		});

		// 鏃犺彍鍗曟暟鎹?		if (Joe.isMobile) {
			!$tocMobileContainer.children().length &&
			$tocMobileContainer.html(
				"<div class=\"toc-nodata\"><em></em>鏆傛棤鐩綍</div>"
			);
		} else {
			!$tocContainer.children().length &&
			$tocContainer.html("<div class=\"toc-nodata\">鏆傛棤鐩綍</div>");
		}

		// 绉诲姩绔痶oc鑿滃崟浜や簰
		if (Joe.isMobile) {
			$btn_mobile_toc.show();
			$btn_mobile_toc.on("click", () => {
				window.sessionStorage.setItem("lastScroll", $html.scrollTop());
				$html.addClass("disable-scroll");
				$mask.addClass("active slideout");
				$mobile_toc.addClass("active");
			});
		}

		$(".toc-container").show();
	},
	/**鍒濆鍖栧乏渚у伐鍏锋潯 */
	initAsideOperate() {
		// 璇勮
		$(".post-operate-comment").on("click", function (e) {
			const $comment = document.querySelector(".joe_comment");
			const $header = document.querySelector(".joe_header");
			if (!$comment || !$header) return;
			e.stopPropagation();
			if (!Boolean(document.querySelector('[id*="comment-"]'))&& !Boolean(document.querySelector("#waline"))) {
				Qmsg.warning("璇勮鍔熻兘涓嶅彲鐢紒");
				return;
			}
			const top = $comment.offsetTop - $header.offsetHeight - 15;
			window.scrollTo({ top });
		});

		// 鍒ゆ柇鏄惁闇€瑕侀殣钘忚彍鍗?		if (Joe.isMobile) return;
		const $asideEl = $(".aside_operations");
		const $operateEl = $(
			".joe_detail__agree,.joe_detail__operate-share,.joe_detail__operate .joe_donate"
		);
		const toggleAsideMenu = (e) => {
			const offsetLeft = $(".joe_post")[0].getBoundingClientRect().left;
			if (offsetLeft < 75) {
				$asideEl.hide();
				$operateEl.show();
			} else {
				$asideEl.show();
				$operateEl.hide();
			}
		};
		toggleAsideMenu();
		window.addEventListener("resize", Utils.throttle(toggleAsideMenu), 500);
	},
	/* 闃呰杩涘害鏉?*/
	initProgress() {
		if (!ThemeConfig.enable_progress_bar) return;
		$(window).off("scroll");
		const progress_bar = $(".joe_progress_bar");
		let win_h, body_h, sHeight;
		const updateProgress = (p) => progress_bar.css("width", p * 100 + "%");
		$(window).on("scroll", function (e) {
			e.stopPropagation();
			win_h = $(window).height();
			body_h = $("body").height();
			sHeight = body_h - win_h;
			window.requestAnimationFrame(function () {
				const perc = Math.max(0, Math.min(1, $(window).scrollTop() / sHeight));
				updateProgress(perc);
			});
		});
	},
	/* 鏂囩珷瑙嗛妯″潡 */
	initVideo() {
		if ($(".joe_detail__article-video").length) {
			const player = $(".joe_detail__article-video .play iframe").attr(
				"data-player"
			);
			$(".joe_detail__article-video .episodes .item").on("click", function (e) {
				e.stopPropagation();
				$(this).addClass("active").siblings().removeClass("active");
				const url = $(this).attr("data-src");
				$(".joe_detail__article-video .play iframe").attr({
					src: player + url,
				});
			});
			$(".joe_detail__article-video .episodes .item").first().click();
		}
	},
	/*璺宠浆鍒版寚瀹氳瘎璁?*/
	async jumpToComment() {
		if (
			ThemeConfig.enable_clean_mode ||
			!ThemeConfig.enable_comment ||
			PageAttrs.metas_enable_comment === "false"
		)
			return;
		const { cid: commentId = "", p: postId = "" } = Utils.getUrlParams();
		if (!commentId) return;
		await Utils.sleep(1000);
		try {
			const commentEl = document.getElementsByTagName("halo-comment");
			if (!commentEl) return;
			const el = $(commentEl[0].shadowRoot.getElementById("halo-comment")).find(
				"#comment-" + commentId
			);
			if (!el) return;
			const offsetTop = el.offset().top - 50;
			// 婊氬姩鍒版寚瀹氫綅缃?			window.scrollTo({ top: offsetTop });
			// 楂樹寒璇ヨ瘎璁哄厓绱?			const el_comment = el.find(".markdown-content").eq(0);
			el_comment.addClass("blink");
			await Utils.sleep(2000);
			el_comment.removeClass("blink");
			// 娓呴櫎url鍙傛暟
			window.history.replaceState(
				null,
				null,
				postId ? `?p=${postId}` : location.origin + location.pathname
			);
			tocbot.refresh();
		} catch (error) {
			console.info(error);
		}
	},
	/* TODO:瀵嗙爜淇濇姢鏂囩珷锛岃緭鍏ュ瘑鐮佽闂?*/
	// initArticleProtect() {
	//   const cid = $(".joe_detail").attr("data-cid");
	//   let isSubmit = false;
	//   $(".joe_detail__article-protected").on("submit", function (e) {
	//     e.preventDefault();
	//     const url = $(this).attr("action") + "&time=" + new Date();
	//     const protectPassword = $(this).find("input[type=\"password\"]").val();
	//     if (protectPassword.trim() === "") return Qmsg.info("璇疯緭鍏ヨ闂瘑鐮侊紒");
	//     if (isSubmit) return;
	//     isSubmit = true;

	// 		Utils.request({
	// 			url: url,
	// 			method: "POST",
	// 			data: {
	//     			cid,
	//     			protectCID: cid,
	//     			protectPassword,
	//     		}
	// 		})
	// 			.then((_res) => {
	//         let arr = [],
	//           str = "";
	//         arr = $(res).contents();
	//         Array.from(arr).forEach((_) => {
	//           if (_.parentNode.className === "container") str = _;
	//         });
	//         if (!/Joe/.test(res)) {
	//           Qmsg.warning(str.textContent.trim() || "");
	//           isSubmit = false;
	//           $(".joe_comment__respond-form .foot .submit button").html(
	//             "鍙戣〃璇勮"
	//           );
	//         } else {
	//           location.reload();
	//         }
	//       }).catch(err=>{
	// 				isSubmit = false;
	// 			});
	//   });
	// },
};

!(function () {
	const omits = ["jumpToComment"];
	document.addEventListener("DOMContentLoaded", function () {
		Object.keys(postContext).forEach(
			(c) =>
				!omits.includes(c) &&
				typeof postContext[c] === "function" &&
				postContext[c]()
		);
	});

	window.addEventListener("load", function () {
		if (omits.length === 1) {
			postContext[omits[0]]();
		} else {
			omits.forEach(
				(c) => c !== "loadingBar" && postContext[c] && postContext[c]()
			);
		}
	});
})();
