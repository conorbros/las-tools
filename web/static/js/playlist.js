M.AutoInit();

function showPortFinishedDiv() {
  const portFinishedDiv = document.getElementById("port-finished-dev");
  portFinishedDiv.style.display = "";
}

function hidePortSelectionDiv() {
  const portSelectionDiv = document.getElementById("port-selection-div");
  portSelectionDiv.style.display = "none";
}

function showPortSelectionDiv() {
  const portSelectionDiv = document.getElementById("port-selection-div");
  portSelectionDiv.style.display = "";
}

function showPortLoadingDiv() {
  const portLoadingDiv = document.getElementById("port-loading-div");
  portLoadingDiv.style.display = "";
}

function hidePortLoadingDiv() {
  const portLoadingDiv = document.getElementById("port-loading-div");
  portLoadingDiv.style.display = "none";
}

function setPlaylistPortComplete(tracksNotFound) {}

export function loading() {
  document.getElementById("port-selection-div").style.display = "none";
  document.getElementById("port-loading-div").style.display = "";
}

function finishedLoading() {
  document.getElementById("port-selection-div").style.display = "";
  document.getElementById("port-loading-div").style.display = "none";
}

document.getElementById("port-button").addEventListener("click", () => {
  const lastFmUsername = document.getElementById("username-textbox").value;
  const songNumber = document.getElementById("song-number-select").value;
  const timePeriod = document.getElementById("time-period-select").value;

  if (!lastFmUsername || !songNumber || !timePeriod) {
    return;
  }

  let data = addSpotifyTokens({
    lastFmUsername,
    songNumber,
    timePeriod,
  });

  loading();

  fetch("/port_toptracks", {
    method: "POST",
    headers: {
      "Content-type": "application/json",
    },
    body: JSON.stringify(data),
  })
    .then((response) => {
      finishedLoading();
      if (response.status === 200) {
        response.json().then((data) => {
          const count = songNumber - data.tracksNotFound.length;
          M.toast({
            html: `${count}/${songNumber} songs were successfully imported.`,
          });
        });
      } else {
        response.text().then(function (text) {
          M.toast({ html: text });
        });
      }
    })
    .catch((error) => {
      finishedLoading();
      M.toast({ html: "There was an internal server error." });
      throw error;
    });
});

function showSpotifyLoginDiv() {
  const spotifyLoginDiv = document.getElementById("spotify-login-div");
  spotifyLoginDiv.style.display = "";
}

function hideSpotifyLoginDiv() {
  const spotifyLoginDiv = document.getElementById("spotify-login-div");
  spotifyLoginDiv.style.display = "none";
}

function SaveDatatoLocalStorage(data) {
  localStorage.setItem("access_token", data.access_token);
  localStorage.setItem("token_type", data.token_type);
  localStorage.setItem("expires_in", data.expires_in);
  localStorage.setItem("refresh_token", data.refresh_token);
  localStorage.setItem("time_obtained", Date.now());
}

/**
 * A code will be sent to the backend if the user finishes spotify login process
 */
async function GetAccessToken() {
  const urlParams = new URLSearchParams(window.location.search);
  const code = urlParams.get("code");
  const error = urlParams.get("error");

  if (code) {
    return fetch(`/get_access_token?code=${code}`)
      .then((response) => response.json())
      .then((data) => {
        if (data.error) {
          window.location = "/playlist";
        } else {
          SaveDatatoLocalStorage(data);
          hideSpotifyLoginDiv();
        }
      })
      .catch((error) => {
        M.toast({ html: "There was a server error logging you into Spotify." });
        throw error;
      });
  }

  if (error) {
    window.alert("Failed to log into Spotify.");
  }
}

function addSpotifyTokens(obj) {
  if (!IsUserLoggedInToSpotify) {
    throw Error("User is not logged in");
  }

  obj.access_token = localStorage.getItem("access_token");
  obj.token_type = localStorage.getItem("token_type");
  obj.expires_in = Number(localStorage.getItem("expires_in"));
  obj.refresh_token = localStorage.getItem("refresh_token");
  obj.time_obtained = Number(localStorage.getItem("time_obtained"));
  return obj;
}

function IsUserLoggedInToSpotify() {
  const access_token = localStorage.getItem("access_token");
  return access_token !== null;
}

async function init() {
  await GetAccessToken();

  if (!IsUserLoggedInToSpotify()) {
    showSpotifyLoginDiv();
  } else {
    showPortSelectionDiv();
  }
}

window.onload = init();
