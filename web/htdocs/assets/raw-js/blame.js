(function f() {
    var body = $('body');
    body.on('mouseenter', '#hashes > a', function(e) {
        var href = $(e.target).attr('href');
        var i = href.indexOf('#');
        var commitHash = href.substring(0, i);
        var cls = 'highlight ' + href.substr(i + 1, 1);
        $('#hashes a[href^="' + commitHash + '"]').addClass(cls);
    });
    body.on('mouseleave', '#hashes > a', function(e) {
        var href = $(e.target).attr('href');
        var i = href.indexOf('#');
        var commitHash = href.substring(0, i);
        var cls = 'highlight ' + href.substr(i + 1, 1);
        $('#hashes a[href^="' + commitHash + '"]').removeClass(cls);
    });
})();
