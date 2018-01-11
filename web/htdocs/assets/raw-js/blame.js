(function f() {
    var body = $('body');
    body.on('mouseenter', '#hashes > a', function(e) {
        var commitHash = $(e.target).attr('href');
        var cls = 'highlight ' + commitHash.substr(commitHash.length - 1);
        $('#hashes a[href="' + commitHash + '"]').addClass(cls);
    });
    body.on('mouseleave', '#hashes > a', function(e) {
        var commitHash = $(e.target).attr('href');
        var cls = 'highlight ' + commitHash.substr(commitHash.length - 1);
        $('#hashes a[href="' + commitHash + '"]').removeClass(cls);
    });
})();
