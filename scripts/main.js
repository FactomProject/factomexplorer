/* init material design/ripple effects */
/*  $.material.init(); */

/* init Bootstrap dropdown */
  $('.dropdown-toggle').dropdown();

$(function() {

  /* Sidr - Init */
    $('#sidr-toggle').sidr({
      name: 'sidr-main',
      source: '#side-navigation',
      renaming: false,
    });

    ///* Sidr - Open click & touch */
    $('#sidr-toggle').on('touchstart click', function () {
        $('.mask').css('display', 'initial');
    });

  /* Sidr - Close */
    $(".mask").swipe ({
      tap: function() {
        $.sidr('close','sidr-main');
        $('.mask').css('display', 'none');
      }
    });

  /* Check if browser width is <= Mobile */
    $(window).bind("load resize",function(e){

      var size = window.getComputedStyle(document.body,':after').getPropertyValue('content');

      if (size >= 'mobile') {

       /* Sidr - Swipe Open/Close if browser width is <= Mobile */
            $(window).swipe("destroy");
          }

          // } else {
          //    $(window).swipe({
          //     swipeLeft: function() {
          //       $.sidr('close', 'sidr-main');
          //       $('.mask').css('display', 'none');
          //     },
          //     swipeRight: function() {
          //       $('.mask').css('display', 'initial');
          //       $.sidr('open', 'sidr-main');
          //     },
          //     preventDefaultEvents: false
          //   });
          // }

        /* Search - Expand on focus if browser width is <= Mobile */
          $('.search-field, .search-type-picker .btn, .clear-search a').focus(function(){
             $('.search').addClass('selected');

            }).blur(function(){
                $('.search').removeClass('selected');
            });

          $('.search-type').focus(function(){
             $('.search').addClass('selected');
             $('.search-field').focus();
            });
        });

/* Make Table Rows Clickable */
$(".clickableRow").click(function() {
            window.document.location = $(this).attr("href");
      });
});

/* Scroll To Top */
$(document).ready(function(){

  // hide #back-top first
  $("#back-top").hide();

  // fade in #back-top
  $(function () {
    $(window).scroll(function () {
      if ($(this).scrollTop() > 400) {
        $('#back-top').fadeIn();
      } else {
        $('#back-top').fadeOut();
      }
    });

    // scroll body to 0px on click
    $('#back-top a').click(function () {
      $('body,html').animate({
        scrollTop: 0
      }, 800);
      return false;
    });
  });

});

/* clear-search */
$('.clear-search a').focus(
  function(){
      $('#searchText').val('');
  });
