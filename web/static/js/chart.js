M.AutoInit();

$(document).ready(function () {
  $("select").formSelect();
  squareSelectValues();
  document.getElementById("square-radio").setAttribute("checked", "");
});

let selectValue = "5x5";

function loading() {
  document.getElementById("last-fm-generate-button").style.display = "none";
  document.getElementById("loader").style.display = "";
}

function finishedLoading() {
  document.getElementById("last-fm-generate-button").style.display = "";
  document.getElementById("loader").style.display = "none";
}

document
  .getElementById("last-fm-generate-button")
  .addEventListener("click", () => {
    const url = new URL("/generate_chart", window.origin);

    let xy = selectValue.split("x");
    let x = Number(xy[0]);
    let y = Number(xy[1]);

    const username = $("#username-textbox").val();

    if (!x || !y || !username) {
      return;
    }

    url.searchParams.append("x", x);
    url.searchParams.append("y", y);
    url.searchParams.append("username", username);

    loading();

    fetch(url, {
      method: "GET",
    })
      .then((response) => {
        finishedLoading();
        if (response.status === 200) {
          response.blob().then((blob) => {
            const url = URL.createObjectURL(blob);
            window.open(url, "_blank");
          });
        } else {
          response.text().then(function (text) {
            M.toast({ html: text });
          });
        }
      })
      .catch((err) => {
        finishedLoading();
        M.toast({ html: "There was an internal server error." });
      });
  });

const squareSizeSelects = ["5x5", "10x10", "20x20", "30x30"];

function squareSelectValues() {
  const data = [
    { id: "5x5", name: "5x5" },
    { id: "10x10", name: "10x10" },
    { id: "20x20", name: "20x20" },
    { id: "30x30", name: "30x30" },
  ];

  let Options = "";
  $.each(data, function (i, val) {
    Options =
      Options + "<option value='" + val.id + "'>" + val.name + "</option>";
  });
  $("#size-select").empty();
  $("#size-select").append(Options);
  $("#size-select").formSelect();

  selectValue = "5x5";
}

function desktopSelectValues() {
  const data = [
    { id: "16x9", name: "16x9" },
    { id: "32x18", name: "32x18" },
  ];

  let Options = "";
  $.each(data, function (i, val) {
    Options =
      Options + "<option value='" + val.id + "'>" + val.name + "</option>";
  });
  $("#size-select").empty();
  $("#size-select").append(Options);
  $("#size-select").formSelect();
  selectValue = "16x9";
}

$("#size-select").on("change", function () {
  selectValue = $(this).val();
});

document.getElementById("square-radio").addEventListener("change", () => {
  squareSelectValues();
});

document.getElementById("desktop-radio").addEventListener("change", () => {
  desktopSelectValues();
});
