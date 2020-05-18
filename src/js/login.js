/*
 * hnsdbc login.js
 * See License
 * ---description---
 *
 */

//---------------- BEGIN MODULE SCOPE VARIABLES --------------
const stateMap = {
  //ローカルキャッシュはここで宣言
  container: undefined
};
//----------------- END MODULE SCOPE VARIABLES ---------------

//------------------- BEGIN UTILITY METHODS ------------------
const encodedString = params => {
  return Object.keys(params)
    .map(key => {
      let val = params[key];
      if (typeof val === "object") val = JSON.stringify(val);
      return [key, encodeURIComponent(val)].join("=");
    })
    .join("&");
};

const makeRequest = opts => {
  //console.info(opts.url);
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open(opts.method, opts.url, true);

    Object.keys(opts.headers).forEach(key => {
      xhr.setRequestHeader(key, opts.headers[key]);
    });
    //console.info(opts.params);
    xhr.send(opts.params);
    xhr.onload = () => {
      //console.info(xhr.status);
      if (xhr.status >= 200 && xhr.status < 302) {
        resolve(JSON.parse(xhr.response));
      } else if ([400, 401, 403, 404].indexOf(xhr.status)) {
        const response = JSON.parse(xhr.response);
        reject(response.error);
      } else {
        //status==500はここでキャッチ
        //console.info(xhr.response);
        reject(xhr.statusText);
      }
    };
    xhr.onerror = () => {
      //console.info(xhr.statusText);
      reject(xhr.statusText);
    };
  });
};

// send formData as itself by using ajax
const ajaxPost = (url, params, token = null) => {
  const headers = {
    "Content-Type": "application/x-www-form-urlencoded; charset=UTF-8"
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return makeRequest({
    method: "POST",
    url: url,
    params: encodedString(params),
    headers: headers
  });
};

// Post a message to the parent(window.postMessage)
const postMessage = msg => {
  // Make sure you are sending a string, and to stringify JSON
  //console.info(msg);
  window.parent.postMessage(msg, "*");
};

//-------------------- END UTILITY METHODS -------------------

//--------------------- BEGIN DOM METHODS --------------------
//---------------------- END DOM METHODS ---------------------

//------------------- BEGIN EVENT HANDLERS -------------------
//Stop when the submit is fired
const onSubmit = event => {
  event.preventDefault();
  const form = new FormData(stateMap.container.querySelector("form"));
  const params = Object.assign(
    ...["email", "password"].map(name => ({ [name]: form.get(name) || "" }))
  );
  const url = "/login";
  ajaxPost(url, params)
    .then(postMessage)
    .catch(error => console.info(error));
};
//-------------------- END EVENT HANDLERS --------------------

//------------------- BEGIN PUBLIC METHODS -------------------

// Begin public method /initModule/
const initModule = () => {
  stateMap.container = document.getElementById("hnsdbc-login");
  // ページ取得
  stateMap.container.innerHTML = template();
  // ローカルイベントのバインド
  stateMap.container.addEventListener("submit", onSubmit, false);
};

//------------------- END PUBLIC METHODS ---------------------

const template = () => {
  return `
<div class="form-wrapper">
  <h1>Sign In</h1>
  <form>
    <div class="form-item">
      <label for="email"></label>
      <input type="email" name="email" required="required" placeholder="Email Address"></input>
    </div>
    <div class="form-item">
      <label for="password"></label>
      <input type="password" name="password" required="required" placeholder="Password"></input>
    </div>
    <div class="button-panel">
      <input type="submit" class="button" title="Sign In" value="Sign In"></input>
    </div>
  </form>
  <div class="form-footer">
    <p><a href="#">Create an account</a></p>
    <p><a href="#">Forgot password?</a></p>
  </div>
</div>`;
};

export { initModule };
