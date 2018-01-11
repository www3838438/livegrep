(function f() {
    $('body').on('mouseenter', '.hashes > a', function(e) {
        commitHash = $(e.target).attr('href');
        $('.hashes a[href="' + commitHash + '"]').addClass('bump');
    });
    $('body').on('mouseleave', '.hashes > a', function(e) {
        commitHash = $(e.target).attr('href');
        $('.hashes a[href="' + commitHash + '"]').removeClass('bump');
    });
})();
