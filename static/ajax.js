function ajax(o) {
    var xhr = new XMLHttpRequest();
    xhr.onreadystatechange = function() {
        if (xhr.readyState == XMLHttpRequest.DONE) {
            if(xhr.status == 200){
                if(o.success != null) {
                    o.success(xhr.response);
                }
            } else {
                if(o.error != null) {
                    o.error();
                }
            }
        }
    }

    var data = new FormData();

    for (var key in o.data) {
        if (o.data.hasOwnProperty(key)) {
            data.append(key, o.data[key]);
        }
    }

    xhr.open(o.method, o.url);
    xhr.send(data);
}