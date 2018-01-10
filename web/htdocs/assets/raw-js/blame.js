(function f() {
    $('body').on('mouseenter', '#prev a', function(e) {
        commitHash = $(e.target).attr('href');
        $('#prev a[href="' + commitHash + '"]').addClass('bump');
    });
    $('body').on('mouseleave', '#prev a', function(e) {
        commitHash = $(e.target).attr('href');
        $('#prev a[href="' + commitHash + '"]').removeClass('bump');
    });
})();
