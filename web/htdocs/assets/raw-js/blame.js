(function f() {
    var body = $('body');

    /* In the file view, highlight the contents of each diff whose
       commit the user mouses over. */

    if (body.hasClass('blamefile')) {
        body.on('mouseenter', '#hashes > a', function(e) {
            var href = $(e.target).attr('href') || "";
            var i = href.indexOf('#');
            if (i == -1) return;
            var commitHash = href.substring(0, i);
            var cls = 'highlight ' + href.substr(i + 1, 1);
            $('#hashes a[href^="' + commitHash + '"]').addClass(cls);
        });
        body.on('mouseleave', '#hashes > a', function(e) {
            var href = $(e.target).attr('href') || "";
            var i = href.indexOf('#');
            if (i == -1) return;
            var commitHash = href.substring(0, i);
            var cls = 'highlight ' + href.substr(i + 1, 1);
            $('#hashes a[href^="' + commitHash + '"]').removeClass(cls);
        });
    }

    /* When the user clicks a hash, remember the line's y coordinate,
       and warp it back to its current location when we land. */

    body.on('click', '#hashes > a', function(e) {
        var y = $(e.currentTarget).offset().top - body.scrollTop();
        Cookies.set("target_y", y, {expires: 1});
        // (Then, let the click proceed with its usual effect.)
    });

    var y = Cookies.get("target_y");
    if (typeof y !== "undefined") {
        Cookies.remove("target_y");
        // body.css("visibility", "hidden");
        $(document).ready(function() {
            // window.setTimeout(function() {
            //     body.css("visibility", "");
            // }, 1);
            var target = $(":target");
            if (target.length) {
                var st = target.offset().top - y;
                if (st > 0)
                    //body.scrollTop(st);
                    window.scroll(0, st);
            }
        });
    }
})();
