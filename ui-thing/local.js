require('jquery');
require('bootstrap-loader');
require('js-cookie');
require('bootstrap-select/dist/js/bootstrap-select.js');
require('bootstrap-select/dist/css/bootstrap-select.css');
require('./local.css');


$(function(){
    var size = $('#repos option').length;
    $('#repos').selectpicker({
        actionsBox: true,
        selectedTextFormat: 'count > 4',
        countSelectedText: '({0} repositories)',
        noneSelectedText: '(all repositories)',
        liveSearch: true,
        width: '20em'
    });
    var c = Cookies.get("repos");
    if (c) {
        var vals = JSON.parse(c);
        if (vals) {
            $("#repos").selectpicker('val',vals);
        }
    }
    $('#repos').change(
        function() {
            var selected = $('#repos').val();
            if (!selected || ($('#repos option').length == selected.length)) {
                Cookies.remove("repos");
            } else {
                Cookies.set("repos",JSON.stringify(selected));
            }
        }
    );
    $('.bootstrap-select .bs-searchbox input').on(
        'keyup',
        function(event) {
            var keycode = (event.keyCode ? event.keyCode : event.which);
            if(keycode == '13'){
                $(this).val("");
                $("#repos").selectpicker('refresh');
            }
        }
    );
    $(window).keyup(function (keyevent) {
        var code = (keyevent.keyCode ? keyevent.keyCode : keyevent.which);
        if (code == 9 && $('.bootstrap-select button:focus').length) {
            $("#repos").selectpicker('toggle');
            $('.bootstrap-select .bs-searchbox input').focus();
        }
    });
});
