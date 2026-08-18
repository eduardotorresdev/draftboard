(function () {
  "use strict";

  var svg = document.getElementById("mundo");
  var camera = document.getElementById("camera");
  var palco = document.getElementById("prancheta");
  var painel = document.getElementById("painel");
  var campos = document.getElementById("campos");

  var mundoL = parseFloat(svg.dataset.largura) || 1;
  var mundoA = parseFloat(svg.dataset.altura) || 1;

  var escala = 1, tx = 0, ty = 0;
  var selecionado = null;

  function aplica() {
    camera.setAttribute("transform", "translate(" + tx + "," + ty + ") scale(" + escala + ")");
  }

  function ajusta() {
    var r = palco.getBoundingClientRect();
    escala = Math.min(r.width / mundoL, r.height / mundoA);
    if (!isFinite(escala) || escala <= 0) { escala = 1; }
    tx = (r.width - mundoL * escala) / 2;
    ty = (r.height - mundoA * escala) / 2;
    aplica();
  }

  function amplia(fator, cx, cy) {
    var novo = Math.min(8, Math.max(0.02, escala * fator));
    tx = cx - (cx - tx) * (novo / escala);
    ty = cy - (cy - ty) * (novo / escala);
    escala = novo;
    aplica();
  }

  // Leva a câmera até um Frame, mantendo-o inteiro na tela com folga.
  function vaiAoFrame(indice) {
    var g = camera.querySelector('.frame[data-frame="' + indice + '"]');
    if (!g) { return; }
    var r = palco.getBoundingClientRect();
    var x = parseFloat(g.dataset.x), y = parseFloat(g.dataset.y);
    var l = parseFloat(g.dataset.l), a = parseFloat(g.dataset.a);
    escala = Math.min(8, Math.min(r.width / (l * 1.4), r.height / (a * 1.4)));
    tx = r.width / 2 - (x + l / 2) * escala;
    ty = r.height / 2 - (y + a / 2) * escala;
    aplica();
    acendeAlvo(indice);
  }

  function acendeAlvo(indice) {
    var alvos = camera.querySelectorAll(".frame.alvo");
    for (var i = 0; i < alvos.length; i++) { alvos[i].classList.remove("alvo"); }
    if (indice === null) { return; }
    var g = camera.querySelector('.frame[data-frame="' + indice + '"]');
    if (g) { g.classList.add("alvo"); }
  }

  function indiceDoFrame(nome) {
    var quadros = camera.querySelectorAll(".frame");
    for (var i = 0; i < quadros.length; i++) {
      if (quadros[i].dataset.nome === nome) { return parseInt(quadros[i].dataset.frame, 10); }
    }
    return null;
  }

  function linha(rotulo, valor, classe) {
    var dt = document.createElement("dt");
    dt.textContent = rotulo;
    var dd = document.createElement("dd");
    dd.textContent = valor;
    if (classe) { dd.className = classe; }
    campos.appendChild(dt);
    campos.appendChild(dd);
    return dd;
  }

  function inspeciona(peca) {
    if (selecionado) { selecionado.classList.remove("selecionado"); }
    selecionado = peca;
    peca.classList.add("selecionado");

    var d = peca.dataset;
    campos.textContent = "";
    linha("caminho", d.caminho);
    linha("camada", d.camada);
    linha("forma", d.forma);
    linha("geometria", d.geo);
    linha("elevação", d.elev);
    linha("tom", d.tom);
    if (d.controle) { linha("controle", d.controle + (d.detalhe ? " " + d.detalhe : "")); }
    if (d.origem) { linha("origem", d.origem); }
    if (d.nota) { linha("nota", d.nota); }
    if (d.para) {
      var dd = linha("ligação", "→ " + d.para, "destino");
      dd.addEventListener("click", function () {
        var i = indiceDoFrame(d.para);
        if (i !== null) { vaiAoFrame(i); }
      });
    }
    painel.hidden = false;
  }

  function fecha() {
    painel.hidden = true;
    if (selecionado) { selecionado.classList.remove("selecionado"); }
    selecionado = null;
    acendeAlvo(null);
  }

  // Pan: arrastar com o botão principal em qualquer lugar do palco.
  //
  // Sem setPointerCapture de propósito: a captura reescreve o alvo do clique
  // seguinte para o elemento que capturou, e aí nenhum clique chegaria na peça
  // que o usuário mirou. Ouvir o resto do arrasto na janela dá o mesmo efeito
  // sem mexer no alvo.
  var arrastando = false, ox = 0, oy = 0, px = 0, py = 0, moveu = false;
  palco.addEventListener("pointerdown", function (ev) {
    if (ev.button !== 0) { return; }
    arrastando = true; moveu = false;
    ox = ev.clientX - tx; oy = ev.clientY - ty;
    px = ev.clientX; py = ev.clientY;
    palco.classList.add("arrastando");
  });
  window.addEventListener("pointermove", function (ev) {
    if (!arrastando) { return; }
    // Uma folga de 4px separa o arrasto do clique trêmulo: mirar uma peça
    // pequena quase sempre move o ponteiro um pixel ou dois.
    if (Math.abs(ev.clientX - px) + Math.abs(ev.clientY - py) > 4) { moveu = true; }
    if (!moveu) { return; }
    tx = ev.clientX - ox; ty = ev.clientY - oy;
    aplica();
  });
  window.addEventListener("pointerup", function () {
    arrastando = false;
    palco.classList.remove("arrastando");
  });

  // Zoom: roda com ctrl/meta (ou pinça do trackpad); roda pura desloca.
  palco.addEventListener("wheel", function (ev) {
    ev.preventDefault();
    var r = palco.getBoundingClientRect();
    if (ev.ctrlKey || ev.metaKey) {
      amplia(Math.exp(-ev.deltaY * 0.01), ev.clientX - r.left, ev.clientY - r.top);
    } else {
      tx -= ev.deltaX; ty -= ev.deltaY;
      aplica();
    }
  }, { passive: false });

  palco.addEventListener("click", function (ev) {
    if (moveu) { return; }
    var peca = ev.target.closest(".peca");
    if (!peca) { fecha(); return; }
    inspeciona(peca);
    if (peca.dataset.para) {
      var i = indiceDoFrame(peca.dataset.para);
      if (i !== null) { vaiAoFrame(i); }
    }
  });

  // Passar o mouse num gatilho acende a Ligação que sai dele e o Frame alvo.
  palco.addEventListener("mouseover", function (ev) {
    var peca = ev.target.closest(".peca.gatilho");
    if (!peca) { return; }
    var caminho = peca.dataset.caminho;
    var ligacoes = camera.querySelectorAll(".ligacao");
    for (var i = 0; i < ligacoes.length; i++) {
      ligacoes[i].classList.toggle("aceso", ligacoes[i].dataset.caminho === caminho);
    }
    var alvo = indiceDoFrame(peca.dataset.para);
    if (alvo !== null) { acendeAlvo(alvo); }
  });
  palco.addEventListener("mouseout", function (ev) {
    if (ev.target.closest(".peca.gatilho")) {
      var ligacoes = camera.querySelectorAll(".ligacao.aceso");
      for (var i = 0; i < ligacoes.length; i++) { ligacoes[i].classList.remove("aceso"); }
    }
  });

  function alterna(acao) {
    var botao = document.querySelector('[data-acao="' + acao + '"]');
    var ligado = botao.getAttribute("aria-pressed") === "true";
    botao.setAttribute("aria-pressed", ligado ? "false" : "true");
    if (acao === "ligacoes") { document.body.classList.toggle("sem-ligacoes", ligado); }
    if (acao === "notas") { document.body.classList.toggle("notas", !ligado); }
  }

  document.querySelector(".teclas").addEventListener("click", function (ev) {
    var botao = ev.target.closest("button");
    if (!botao) { return; }
    var r = palco.getBoundingClientRect();
    switch (botao.dataset.acao) {
      case "mais": amplia(1.25, r.width / 2, r.height / 2); break;
      case "menos": amplia(0.8, r.width / 2, r.height / 2); break;
      case "ajustar": ajusta(); break;
      default: alterna(botao.dataset.acao);
    }
  });

  document.getElementById("fecha").addEventListener("click", fecha);

  document.addEventListener("keydown", function (ev) {
    if (ev.key === "0") { ajusta(); }
    if (ev.key === "l") { alterna("ligacoes"); }
    if (ev.key === "n") { alterna("notas"); }
    if (ev.key === "Escape") { fecha(); }
    if (ev.key === "+" || ev.key === "=") {
      var a = palco.getBoundingClientRect();
      amplia(1.25, a.width / 2, a.height / 2);
    }
    if (ev.key === "-") {
      var b = palco.getBoundingClientRect();
      amplia(0.8, b.width / 2, b.height / 2);
    }
  });

  window.addEventListener("resize", ajusta);
  ajusta();
})();
