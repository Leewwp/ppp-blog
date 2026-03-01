/**鏂囩珷椤甸€昏緫 */
const journalContext = {
	/* 婵€娲诲垪琛ㄧ壒鏁?*/
	initEffect() {
		$(".joe_loading").remove();
		$(".joe_journals__list").removeClass("hidden");
		if (!ThemeConfig.enable_journal_effect) return;
		new WOW({
			boxClass: "wow",
			animateClass: ThemeConfig.journal_list_effect_class || "fadeIn",
			offset: 0,
			mobile: true,
			live: true,
		}).init();
	},

	/* 鏃ュ織鍙戝竷鏃堕棿鏍煎紡鍖?*/
	formatTime() {
		const $allJournalTime = $(".joe_journal-posttime");
		$allJournalTime.each(function () {
			const $this = $(this);
			$this.html(Utils.timeAgo($this.text()));
		});
	},
	/* 鐐硅禐 */
	initLike() {
		if (!ThemeConfig.enable_like_journal) return;
		const $allItems = $(".joe_journal__item");
		if ($allItems.length) {
			$allItems.each(function (_, item) {
				const $this = $(item);
				const cid = $this.attr("data-cid");
				const viewer = ($this.attr("data-viewer") || "anonymousUser").trim();
				const agreeStoreKey = encryption(`agree-journal:${viewer}`);
				let likes = +($this.attr("data-clikes") || 0);
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
				const $iconLike = $this.find(".journal-like");
				const $iconUnlike = $this.find(".journal-unlike");
				const $likeNum = $this.find(".journal-likes-num");
				const renderLikeState = () => {
					if (flag) {
						$iconLike.hide();
						$iconUnlike.show();
					} else {
						$iconLike.show();
						$iconUnlike.hide();
					}
					$likeNum.html(likes);
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
							group: "moment.halo.run",
							plural: "moments",
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
			});
		}
	},
	/* 璇勮鍙婃姌鍙?*/
	initComment() {
		if (ThemeConfig.enable_clean_mode || !ThemeConfig.enable_comment_journal)
			return;
		$(".journal_comment_expander,.journal-comment").on("click", function (e) {
			e.stopPropagation();
			const $this = $(this);
			const $parent = $this.parents(".footer-wrap");
			// const compComment = $parent.find("halo-comment")[0]._wrapper.$refs.inner;
			// 灞曞紑鍔犺浇璇勮
			// if (!$parent.hasClass("open")) {
			// 	return;
			// }
			console.log("ping")
			$parent.toggleClass("open");
			$parent
				.find(".journal_comment_expander_txt")
				.html(($parent.hasClass("open") ? "鏀惰捣" : "鏌ョ湅") + "璇勮");
		});
	},
	/* 鍐呭鎶樺彔/灞曞紑 */
	initExpander() {
		$(".journal_content_expander i").on("click", function (e) {
			e.stopPropagation();
			$(this).parents(".joe_journal_body").toggleClass("open");
		});
	},
	/* 鏃ュ織鍧楁姌鍙?*/
	foldBlock() {
		const $allBlocks = $(".joe_journal_body .content-wrp");
		$allBlocks.each(function () {
			const $this = $(this);
			if (
				$this[0].getBoundingClientRect().height >=
        ThemeConfig.journal_block_height
			) {
				$this.siblings(".journal_content_expander").show();
			}
		});
	},
};

!(function () {
	const omits = ["foldBlock"];
	document.addEventListener("DOMContentLoaded", function () {
		Object.keys(journalContext).forEach(
			(c) => !omits.includes(c) && journalContext[c]()
		);
	});
	window.addEventListener("load", function () {
		journalContext.foldBlock();
	});
})();
