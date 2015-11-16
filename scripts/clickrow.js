$(document).ready(function(){
            $('.datadisplay').css({
                        "max-width": "60%",
                        "text-overflow": "ellipsis",
                        "word-wrap": "normal",
                        //backgroundColor: "#ffe",
                        "overflow": "hidden"
                    });

    
            var currWrap = "normal";
            $('.datadisplay').click(function () {
                currWrap = $( this ).css( "word-wrap");
                if (currWrap === "break-word") {
                    $( this ).css({
                        "max-width": "60%",
                        "text-overflow": "ellipsis",
                        "word-wrap": "normal",
                        "overflow": "hidden"
                    });
                } else {
                    $( this ).css({
                        "word-wrap": "break-word",
                        "max-width": "98%",
                        "white-space": "normal"
                    });
                }
            });

        //$('.hasicon').css({backgroundColor: "#ffe", borderLeft: "5px solid #ccc" });
        
        $('.hasicon > span').on('click', function() {
            expandCell = $(this).closest('pre');
            isExpanded = expandCell.attr("data-expanded");
            if(isExpanded === "true"){
                expandCell.attr("data-expanded", "false");
                expandCell.css({
                    "height": "6em",
                    "overflow": "hidden"
                });
            } else {
                expandCell.attr("data-expanded", "true");
                expandCell.css({
                    "height": "auto",
                    "border": "1px solid #ccc"
                });
            }
        });
            
});

document.getElementById("rJSON").onclick = function() {
    thisPre = this.childNodes[0];
    if (thisPre.style.height === 'auto') {
        thisPre.style.height = '6em';
        thisPre.style.overflow = 'hidden';
    } else {
        thisPre.style.height = 'auto';
        thisPre.style.border = '1px solid #ccc';
        thisPre.style.borderRadius = "4px";
    }
}

