let juegos = [];

const formJuego = document.getElementById("form-juego");
const juegoTitulo = document.getElementById("juego-titulo");

const formMapa = document.getElementById("form-mapa");
const mapaTitulo = document.getElementById("mapa-titulo");
const selectJuegoMapa = document.getElementById("select-juego-mapa");

const selectJuegoRandom = document.getElementById("select-juego-random");
const listaMapas = document.getElementById("lista-mapas");

const btnMapaAleatorio = document.getElementById("btn-mapa-aleatorio");
const resultadoAleatorio = document.getElementById("resultado-aleatorio");
const btnReiniciarBaneos = document.getElementById("btn-reiniciar-baneos");

// ------------------- Funciones -------------------

async function cargarJuegos() {

  try {
    const res = await fetch("/api/juegos");
    console.log("Llego la respuesta de /api/juegos:", res);
    const data = await res.json();
    juegos = data;
    actualizarSelects();
    mostrarMapas();
  } catch (err) {
    console.error("Error cargando juegos:", err);
  }
}

function actualizarSelects() {
  [selectJuegoMapa, selectJuegoRandom].forEach(select => {
    const selectedValue = select.value;
    select.innerHTML = '<option value="">-- Selecciona un juego --</option>';
    juegos.forEach(juego => {
      const option = document.createElement("option");
      option.value = juego.id;
      option.textContent = juego.titulo;
      select.appendChild(option);
    });
    if (selectedValue) select.value = selectedValue;
  });

  console.log("Selects actualizados.");
}

function mostrarMapas() {
  listaMapas.innerHTML = "";
  const juegoId = selectJuegoRandom.value;
  if (!juegoId) return;

  const juego = juegos.find(j => j.id == juegoId);
  if (!juego) return;

  juego.mapas.forEach(mapa => {
    const li = document.createElement("li");
    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.checked = mapa.baneado || false;

    checkbox.addEventListener("change", async () => {
      mapa.baneado = checkbox.checked;
      // Guardar el cambio en la DB
      try {
        await fetch("/api/mapas/baneo", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ mapaId: mapa.id, baneado: checkbox.checked })
        });
      } catch (err) {
        console.error("Error actualizando baneado:", err);
      }
    });

    li.appendChild(checkbox);
    li.append(` ${mapa.titulo}`);
    listaMapas.appendChild(li);
  });
}

// ------------------- Eventos -------------------

formJuego.addEventListener("submit", async e => {
  e.preventDefault();
  const titulo = juegoTitulo.value.trim();
  if (!titulo) return;

  try {
    const res = await fetch("/api/juegos", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ titulo })
    });
    const nuevoJuego = await res.json();
    juegos.push({ ...nuevoJuego, mapas: [] });
    juegoTitulo.value = "";
    actualizarSelects();
  } catch (err) {
    console.error("Error agregando juego:", err);
  }
});

formMapa.addEventListener("submit", async e => {
  e.preventDefault();
  const titulo = mapaTitulo.value.trim();
  const juegoId = parseInt(selectJuegoMapa.value);
  if (!titulo || !juegoId) return;

  try {
    const res = await fetch("/api/mapas", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ titulo, juegoId })
    });
    const nuevoMapa = await res.json();

    const juego = juegos.find(j => j.id == juegoId);
    if (juego) {
      juego.mapas.push({ ...nuevoMapa, baneado: false });
      if (selectJuegoRandom.value == juegoId) mostrarMapas();
    }

    mapaTitulo.value = "";
  } catch (err) {
    console.error("Error agregando mapa:", err);
  }
});

selectJuegoRandom.addEventListener("change", mostrarMapas);

btnReiniciarBaneos.addEventListener("click", async () => {
  const juegoId = selectJuegoRandom.value;
  if (!juegoId) return;

  const juego = juegos.find(j => j.id == juegoId);
  if (!juego) return;

  try {
    await Promise.all(
      juego.mapas.map(mapa =>
        fetch("/api/mapas/baneo", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ mapaId: mapa.id, baneado: false })
        })
      )
    );

    juego.mapas.forEach(mapa => (mapa.baneado = false));
    mostrarMapas();
  } catch (err) {
    console.error("Error reiniciando baneos:", err);
  }
});

btnMapaAleatorio.addEventListener("click", () => {
  const juegoId = selectJuegoRandom.value;
  if (!juegoId) {
    resultadoAleatorio.textContent = "Selecciona un juego primero.";
    return;
  }

  const juego = juegos.find(j => j.id == juegoId);
  if (!juego) return;

  const mapasDisponibles = juego.mapas.filter(m => !m.baneado);
  if (mapasDisponibles.length === 0) {
    resultadoAleatorio.textContent = "No hay mapas disponibles (todos baneados).";
    return;
  }

  const aleatorio = mapasDisponibles[Math.floor(Math.random() * mapasDisponibles.length)];
  resultadoAleatorio.textContent = `Mapa seleccionado: ${aleatorio.titulo}`;
});

// ------------------- Inicialización -------------------
cargarJuegos();
